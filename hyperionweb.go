// Provides a web interface for managing hyperion
// Can do things with hyperion when you come home or go to bed

// To use run "go run hyperionweb.go <path to index.html>"
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "html/template"
    "log"
    "net"
    "net/http"
    "os"
    "bufio"
    "strconv"
    "time"
    "strings"
    "io/ioutil"
    "github.com/jasonlvhit/gocron"
    "golang.org/x/crypto/ssh"
)

const (
    // change these to your liking
    HYPERION_IP         = "192.168.0.120"
    WEB_UI_HOST_PORT    = "1234"
    PRIORITY            = "0"
    SEARCHFOR           = "A0:B1:C2:D3:E4:F5"
    ROUTER_HOST         = "192.168.0.1:22"
    USER                = "myUser"

    JSON_PORT           = "19444"
    SSH_PORT            = "22"
    HYPERION_SERVER     = HYPERION_IP+":"+JSON_PORT
    HYPERION_HOST       = HYPERION_IP+":"+SSH_PORT
)

type Args struct {
    FadeFactor float64 `json:"fadeFactor"`
    Speed      float64 `json:"speed"`
}

type Effect struct {
    Args   Args   `json:"args"`
    Name   string `json:"name"`
    script string `json:"script"`
}

type Priority struct {
    Priority uint `json:"priority"`
}

type Transform struct {
    Blacklevel     [3]float64 `json:"blacklevel"`
    Gamma          [3]float64 `json:"gamma"`
    Id             string     `json:"id"`
    SaturationGain float64    `json:"saturationGain"`
    Threshold      [3]float64 `json:"threshold"`
    ValueGain      float64    `json:"valueGain"`
    Whitelevel     [3]float64 `json:"whitelevel"`
}

type ServerInfo struct {
    Effects    []Effect    `json:"effects"`
    Priorities []Priority  `json:"priorities"`
    Transform  []Transform `json:"transform"`
}

type ServerInfoWrapper struct {
    Info    ServerInfo `json:"info"`
    Success bool       `json:"success"`
}

type ColorMap struct {
    Name  string
    Value rgb
}

var mappedColors []ColorMap

type rgb struct {
    r int
    g int
    b int
}

var serverInfo ServerInfo
var lastColor rgb = rgb{ r: -1, g: -1, b: -1 }
var lastValue float64
var isClear bool = false
var effectRunning bool = false
var hasBeenHome bool
var publicKey []byte

var SERVERPATH string

func getServerInfo() (ServerInfo, error) {
    var info ServerInfoWrapper
    conn, err := net.Dial("tcp", HYPERION_SERVER)
    if err != nil {
        return serverInfo, err
    }

    fmt.Fprint(conn, `{"command":"serverinfo"}`+"\n")

    line, _, err := bufio.NewReader(conn).ReadLine()
    if err != nil {
        return serverInfo, err
    }

    err = json.Unmarshal(line, &info)
    serverInfo = info.Info
    return serverInfo, err
}

func loadColors() {
    // colors.txt
    file, err := os.Open(SERVERPATH+"/js/colors.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() { //for all lines
        line := scanner.Text()
        line = strings.ToLower(line) //lowercased

        index := strings.LastIndex(line, " ") //index to last space
        blue := line[index+1:len(line)] //from index+1 to end of line
        //line = from 0 to last space (blue is gone from line, trim right to remove all spaces
        line = strings.TrimRight(line[0:index], " ")

        index = strings.LastIndex(line, " ")
        green := line[index+1:len(line)]
        line = strings.TrimRight(line[0:index], " ")

        index = strings.LastIndex(line, " ")
        red := line[index+1:len(line)]
        name := strings.TrimRight(line[0:index], " ")

        r, _ := strconv.Atoi(red)
        g, _ := strconv.Atoi(green)
        b, _ := strconv.Atoi(blue)
        color := rgb {r: r, g: g, b: b}
        colorMap := ColorMap{Name: name, Value: color}

        mappedColors = append(mappedColors, colorMap) //add to list
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}

func sendToHyperion(s string) (string, error) {
    conn, err := net.Dial("tcp", HYPERION_SERVER)
    if err != nil {
        return "", err
    }
    fmt.Fprint(conn, s)
    var response [1024]byte
    n, err := conn.Read(response[:])
    if err != nil {
        return "", err
    }
    return string(response[:n]), nil
}

func hypStaticColor(r, g, b string) string {
    return fmt.Sprintf(`{"color":[%s,%s,%s],"command":"color","priority":%s}`+"\n",
        r, g, b, PRIORITY)
}

func hypStructStaticColor(color rgb) string {
    return fmt.Sprintf(`{"color":[%d,%d,%d],"command":"color","priority":%s}`+"\n",
        color.r, color.g, color.b, PRIORITY)
}

func hypValueGain(n string) string {
    return fmt.Sprintf(`{"command":"transform","transform":{"valueGain":%s}}`+"\n", n)
}

func hypEffect(n string) string {
    return fmt.Sprintf(`{"command":"effect","effect":{"name":"%s"},"priority":%s}`+"\n",
        n, PRIORITY)
}

func hypColor(n string) string {
    return fmt.Sprintf(`{"command":"color","color":{"name":"%s"},"priority":%s}`+"\n",
        n, PRIORITY)
}

func hypClear() string {
    return `{"command":"clear","priority":0}` + "\n"
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")

    var err error
    var t *template.Template
    var b bytes.Buffer
    var response = sshCommand("systemctl status hyperion", HYPERION_HOST)

    if strings.Contains(response, "active (running)") {
        serverInfo, err = getServerInfo()
        if err != nil {
            http.Error(w, err.Error(), 500)
        } else {
            t, err = template.ParseFiles(SERVERPATH + "index.html")
            if err != nil {
                panic("index.html not found")
            }

            err = t.Execute(&b, serverInfo)
        }
    } else {
        t, err = template.ParseFiles(SERVERPATH + "stopped.html")
        if err != nil {
            panic("stopped.html not found")
        }

        err = t.Execute(&b, struct{}{})
    }

    if err == nil {
        fmt.Fprint(w, b.String())
    } else {
        http.Error(w, err.Error(), 500)
    }
}

func handlerColorName(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")

    colorName := strings.ToLower(r.PostFormValue("colorName"))

    // find rgb values
    for i := 0; i < len(mappedColors); i++ {
        colorMap := mappedColors[i]
        if colorMap.Name == colorName {
            lastColor = colorMap.Value

            var resp string
            hyperionResp, err := sendToHyperion(hypStructStaticColor(lastColor))
            if err != nil {
                resp = err.Error()
            } else {
                resp = fmt.Sprintf("%s", hyperionResp)
                log.Printf("Setting the color to: %s (%d %d %d)",
                    colorName, lastColor.r, lastColor.g, lastColor.b)
                isClear = false;
                fmt.Fprint(w, resp)
                return
            }
        }
    }
    log.Printf("Could not find color name %s", colorName)

}

func handlerStaticColor(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    red, green, blue := r.PostFormValue("r"), r.PostFormValue("g"), r.PostFormValue("b")

    ri, _ := strconv.Atoi(red)
    gi, _ := strconv.Atoi(green)
    bi, _ := strconv.Atoi(blue)

    lastColor = rgb{ r: ri, g: gi, b: bi }

    var resp string
    hyperionResp, err := sendToHyperion(hypStaticColor(red, green, blue))
    if err != nil {
        resp = err.Error()
    } else {
        resp = fmt.Sprintf("%s", hyperionResp)
        log.Printf("Setting the color to: %d %d %d", lastColor.r, lastColor.g, lastColor.b)
        isClear = false
        effectRunning = false
    }
    fmt.Fprint(w, resp)
}

func handlerValueGain(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    valueGain := r.PostFormValue("valueGain")

        // string to float
        i, err := strconv.ParseFloat(valueGain, 3)
        if err != nil {
            fmt.Println(err)
            return
        }

        p := i * 10
        percentage := strconv.FormatFloat(i, 'f', 0, 32)
        i = i / 100 // want numbers between [0-1]
        valueGain = strconv.FormatFloat(i, 'f', 2, 32) // float to string
        //fmt.Printf("ValueGain is: %s", valueGain)

        var resp string
        hyperionResp, err := sendToHyperion(hypValueGain(valueGain))

        if err != nil {
            resp = err.Error()
        } else {
            resp = fmt.Sprintf("%s", hyperionResp)
            log.Printf("Setting the value to %s%%", percentage)
            lastValue = p / 10;
        }
        fmt.Fprint(w, resp)
}

func handlerEffect(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    effectName := r.PostFormValue("effect")
    var resp string
    hyperionResp, err := sendToHyperion(hypEffect(effectName))
    if err != nil {
        resp = err.Error()
    } else {
        resp = fmt.Sprintf("%s", hyperionResp)
        isClear = false
        effectRunning = true
        log.Printf("Choosed the effect %s", effectName)
    }
    fmt.Fprint(w, resp)
}

func handlerClear(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    clear := r.PostFormValue("clear")


    var resp string
    if clear != "clear" {
        resp = "<code>NOPE</code>"
    } else {
        hyperionResp, err := sendToHyperion(hypClear())
        if err != nil {
            resp = err.Error()
        } else {
            resp = fmt.Sprintf("%s", hyperionResp)
            isClear = true
            effectRunning = false
            log.Printf("Cleared all priroity channels")
        }
    }
    fmt.Fprint(w, resp)
}

func handlerEffectList(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")

    var resp string
    var effect Effect
    resp = "{\"effects\":["

    for i := 0; i < len(serverInfo.Effects); i++ {
        effect = serverInfo.Effects[i]
        resp += "{\"name\": \"" + effect.Name + "\"}"

        if i < len(serverInfo.Effects)-1 {
            resp += ", "
        }
    }

    resp += "]}"

    fmt.Fprint(w, resp)
}

func handlerGetValueGain(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")

    var resp = fmt.Sprintf("{\"valueGain\": \"%f\"}", lastValue)
    fmt.Fprint(w, resp)
}

func handlerExists(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")

    var resp = "{\"success\": \"true\"}"
    fmt.Fprint(w, resp)
}

func handlerRestart(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    restart := r.PostFormValue("restart")

    var resp string
    if restart != "restart" {
        resp = "<code>NOPE</code>"
    } else {
        sshCommand("systemctl restart hyperion", HYPERION_HOST)
        resp = "<code class='feedback'>Restarted hyperion</code>"
        isClear = true;
        log.Printf("Restarted hyperion")
        }
    fmt.Fprint(w, resp)
}

func handlerStart(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    start := r.PostFormValue("start")

    var resp string
    if start != "start" {
        resp = "<code>NOPE</code>"
    } else {
        sshCommand("systemctl start hyperion", HYPERION_HOST)
        resp = "<code class='feedback'>Started hyperion</code>"
        isClear = true;
        log.Printf("Started hyperion")
        }
    fmt.Fprint(w, resp)
}

func handlerStop(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html")
    stop := r.PostFormValue("stop")

    var resp string
    if stop != "stop" {
        resp = "<code>NOPE</code>"
    } else {
        sshCommand("systemctl stop hyperion", HYPERION_HOST)
        resp = "<code class='feedback'>Stopped hyperion</code>"
        isClear = true;
        log.Printf("Stopped hyperion")
        }
    fmt.Fprint(w, resp)
}

func sshCommand(cmd string, sshAdress string) string {
    signer, err := ssh.ParsePrivateKey(publicKey)
    if err != nil {
        log.Fatalf("parse key failed:%v", err)
    }
    // An SSH client is represented with a ClientConn. Currently only
    // the "password" authentication method is supported.
    //
    // To authenticate with the remote server you must pass at least one
    // implementation of AuthMethod via the Auth field in ClientConfig.
    config := &ssh.ClientConfig{
        User: "root",
        Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
    }

    client, err := ssh.Dial("tcp", sshAdress, config)
    if err != nil {
        log.Printf("SSH: Failed to dial. %s", err.Error())
    }

    // Each ClientConn can support multiple interactive sessions,
    // represented by a Session.
    session, err := client.NewSession()
    if err != nil {
        log.Printf("SSH: Failed to create session. %s", err.Error())
    }
    defer session.Close()

    // Once a Session is created, you can execute a single command on
    // the remote side using the Run method.
    var b bytes.Buffer
    session.Stdout = &b

    err = session.Run(cmd);
    ee, _ := err.(*ssh.ExitError)

    if err != nil && ee.Waitmsg.ExitStatus() != 3 {
        log.Printf("SSH: Failed to run. %s", err.Error())

}
    //fmt.Println("%s", b.String()) //debugging
    return b.String()
}

func isDeviceHome() bool {
    if (strings.Contains(sshCommand("nmap -sn 192.168.0.96", ROUTER_HOST), SEARCHFOR) ||
        strings.Contains(sshCommand("/etc/config/show_wifi_clients.sh",
            ROUTER_HOST), strings.ToLower(SEARCHFOR))) {
        hasBeenHome = true
        //log.Printf("Found " + SEARCHFOR)
        return true
    } else {
        //log.Printf("Nope, couldn't find any " + SEARCHFOR)
        return false
    }
}

func autoON() {
    hasBeenHome = false
    for {
        for time.Now().Hour() >= 16 && time.Now().Hour() <= 22 {

            isDeviceHome := isDeviceHome()

            log.Printf("autoON:\tisDeviceHome: %t\t!isClear: %t\t!effectRunning: %t\tlastColor: %d",
                isDeviceHome, !isClear, !effectRunning, lastColor)

            // automatic turn on
            if (isDeviceHome && !isClear && !effectRunning && lastColor == rgb{ r: 0, g: 0, b: 0 }) {
                newColor := rgb{ r: 255, g: 111, b: 3 }
                sendToHyperion(hypStructStaticColor(newColor))
                sendToHyperion(hypValueGain("1"))

                lastColor = newColor
                log.Printf("autoON: Setting the color to: %d %d %d", lastColor.r, lastColor.g, lastColor.b)
                isClear = false
                return
            } else {
                time.Sleep(30 * time.Second) // sleep for a bit
            }
        }
        time.Sleep(30 * time.Second)
    }
}


func autoOFF() {
    for {
        for time.Now().Hour() >= 22 || time.Now().Hour() <= 5 {

            isDeviceHome := isDeviceHome()

            log.Printf("autoOFF:\t!isDeviceHome: %t\t!isClear: %t\thasBeenHome: %t\tlastColor: %d",
                !isDeviceHome, !isClear, hasBeenHome, lastColor)

            // automatic turn off
            if (!isDeviceHome && !isClear && hasBeenHome && lastColor != rgb{ r: 0, g: 0, b: 0 }) {
                newColor := rgb{ r: 0, g: 0, b: 0 }
                sendToHyperion(hypStructStaticColor(newColor))

                lastColor = newColor
                log.Printf("autoOFF: Setting the color to: %d %d %d", lastColor.r, lastColor.g, lastColor.b)
                isClear = false
                hasBeenHome = false
                return
            } else {
                time.Sleep(30 * time.Second) // sleep for a bit
            }
        }
        time.Sleep(30 * time.Second)
    }
}

func main() {
    // get ssh public key, goes in main to avoid multiple open files.
    var err error
    publicKey, err = ioutil.ReadFile("/home/" + USER + "/.ssh/id_rsa")
    if err != nil {
        log.Fatal(err)
    }

    /*
    go autoON()
    //go autoOFF() // not really wanted since phone disconnects from wifi after 10 minutes.

    // launch autoON every day at 16:00
    gocron.Every(1).Day().At("16:00").Do(autoON)
    // remove, clear and next_run
    _, time := gocron.NextRun()
    fmt.Println(time)

    gocron.Remove(autoON)

    // function Start start all the pending jobs
    gocron.Start()
    */

    // Get current working directory
    pwd, err := os.Getwd()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    if len(os.Args) != 2 {
        log.Printf("Usage:\n" + pwd + "/hyperionweb.go /path/to/index")
        return
    }

    SERVERPATH = os.Args[1]
    if SERVERPATH[len(SERVERPATH)-1] != '/' {
        SERVERPATH += "/"
    }

    serverInfo, err = getServerInfo()

    loadColors()
    lastValue = serverInfo.Transform[0].ValueGain

    http.HandleFunc("/", handlerRoot)
    http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir(SERVERPATH+"/css"))))
    http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir(SERVERPATH+"/js"))))
    http.HandleFunc("/set_color_name", handlerColorName)
    http.HandleFunc("/set_static", handlerStaticColor)
    http.HandleFunc("/set_value_gain", handlerValueGain)
    http.HandleFunc("/set_effect", handlerEffect)
    http.HandleFunc("/get_value_gain", handlerGetValueGain)
    http.HandleFunc("/get_effect_list", handlerEffectList)
    http.HandleFunc("/host_exists", handlerExists)
    http.HandleFunc("/do_clear", handlerClear)
    http.HandleFunc("/do_restart", handlerRestart)
    http.HandleFunc("/do_start", handlerStart)
    http.HandleFunc("/do_stop", handlerStop)

    // Verify that index.html exists
    if _, err := os.Stat(SERVERPATH+"/index.html"); err == nil {
        log.Println("Establishing server on port:", WEB_UI_HOST_PORT)
        err := http.ListenAndServe(":"+WEB_UI_HOST_PORT, nil)
        if err != nil {
            log.Fatal(err)
        }
    }
    log.Println("Failed to find index.html")
}
