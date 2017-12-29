package main

import (
    "github.com/influxdata/influxdb/client"
    "github.com/AlekSi/zabbix-sender"
    "github.com/pelletier/go-toml"
    "fmt"
    "net"
    "log"
    "os"
    "bufio"
    "os/exec"
    "net/url"
    "strings"
    "time"
    "strconv"
)

func herr(err error) {
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func perr(err error) {
    if err != nil {
        fmt.Println(err)
    }
}

func slashSplitter(c rune) bool {
    return c == '/'
}

type host_map struct {
    host    string
    alias    string
    src_host  string
    group   string
}

var hostMap = map[string]host_map{}

func readPoints(config *toml.Tree, con *client.Client) {
    args := []string{"-B 1", "-D", "-r0", "-O 0", "-Q 10", "-p 1000", "-l"}
    groups := config.Get("hosts").(*toml.Tree)
    for _, g := range groups.Keys() {
        sub := groups.Get(g).([]interface{})
        for _,h := range sub{
            host  := h.([]interface{})[0].(string)
            alias := h.([]interface{})[1].(string)
            args = append(args, host)
            hostMap[host] = host_map{
                host: host,
                alias: alias,
                group: g,
            }
        }
        log.Printf("Going to ping the group:%s following hosts: %q", g, sub)
    }
    fping := config.Get("main.fping_location").(string)
    cmd := exec.Command(fping, args...)
    stdout, err := cmd.StdoutPipe()
    herr(err)
    stderr, err := cmd.StderrPipe()
    herr(err)
    cmd.Start()
    perr(err)

    buff := bufio.NewScanner(stderr)
    for buff.Scan() {
        text := buff.Text()
        fields := strings.Fields(text)
        // Ignore timestamp
        if len(fields) > 1 {
            host := fields[0]
            group := hostMap[host].group
            alias := hostMap[host].alias
            data := fields[4]
            dataSplitted := strings.FieldsFunc(data, slashSplitter)
            // Remove ,
            dataSplitted[2] = strings.TrimRight(dataSplitted[2], "%,")
            sent, recv, lossp := dataSplitted[0], dataSplitted[1], dataSplitted[2]
            min, max, avg := "", "", ""
            // Ping times
            if len(fields) > 5 {
                times := fields[7]
                td := strings.FieldsFunc(times, slashSplitter)
                min, avg, max = td[0], td[1], td[2]
            }
            log.Printf("Host:%s, group:%s, loss: %s, min: %s, avg: %s, max: %s", host, group, lossp, min, avg, max)
            writePoints(config, con, host, alias, group, sent, recv, lossp, min, avg, max)
        }
    }
    std := bufio.NewReader(stdout)
    line, err := std.ReadString('\n')
    perr(err)
    log.Printf("stdout:%s", line)
}

func writePoints(config *toml.Tree, con *client.Client, host string, alias string, group string, sent string, recv string, lossp string, min string, avg string, max string) {
    db := config.Get("influxdb.db").(string)
    src_host := config.Get("main.src_host").(string)
    loss, _ := strconv.Atoi(lossp)
    limit_loss, _ := config.Get("alerts.loss_limit").(int)
    alert_provider := config.Get("alerts.provider").(string)
    alert_dst := config.Get("alerts.dst").(string)
    alert_key := config.Get("alerts.key").(string)
    alert_msg := fmt.Sprintf("LOSS: [%s][%s] - %d", alias, host, loss)
    if loss > limit_loss {
        if alert_provider == "zabbix_sender"{
            alert_data := map[string]interface{}{alert_key: alert_msg}
            di := zabbix_sender.MakeDataItems(alert_data, src_host)
            addr, _ := net.ResolveTCPAddr("tcp", alert_dst)
            res, _ := zabbix_sender.Send(addr, di)
            log.Print(res)
        }
    }
    pts := make([]client.Point, 1)
    tags := map[string]string{}
    tags = map[string]string{
        "host": host,
        "alias": alias,
        "src_host": src_host,
        "group": group,
    }
    fields := map[string]interface{}{}
    if min != "" && avg != "" && max != "" {
        min, _ := strconv.ParseFloat(min, 64)
        avg, _ := strconv.ParseFloat(avg, 64)
        max, _ := strconv.ParseFloat(max, 64)
        fields = map[string]interface{}{
                "loss": loss,
                "min": min,
                "avg": avg,
                "max": max,
        }
    } else {
        fields = map[string]interface{}{
                "loss": loss,
        }
    }
    pts[0] = client.Point{
        Measurement: config.Get("influxdb.measurement").(string),
        Tags: tags,
        Fields: fields,
        Time: time.Now(),
        Precision: "",
    }

    bps := client.BatchPoints{
        Points:          pts,
        Database:        db,
        RetentionPolicy: "autogen",
    }
    _, err := con.Write(bps)
    if err != nil {
        log.Fatal(err)
    }
}

func main() {
    config, err := toml.LoadFile("config.toml")
    if err != nil {
        fmt.Println("Error:", err.Error())
        os.Exit(1)
    }

    influx_url := config.Get("influxdb.url").(string)
    username   := config.Get("influxdb.user").(string)
    password   := config.Get("influxdb.pass").(string)

    u, err := url.Parse(influx_url)
    if err != nil {
        log.Fatal(err)
    }

    conf := client.Config{
        URL:      *u,
        Username: username,
        Password: password,
    }

    con, err := client.NewClient(conf)
    if err != nil {
        log.Fatal(err)
    }

    dur, ver, err := con.Ping()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Connected to influxdb! %v, %s", dur, ver)

    readPoints(config, con)
}
