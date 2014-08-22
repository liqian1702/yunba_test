package yunba_test

import (
    "net/http"
    "encoding/json"
    "io"
    "io/ioutil"
    "fmt"
    "strings"
    "strconv"
    "time"
    // "os"
    "testing"

    UUID "code.google.com/p/go-uuid/uuid"
    MQTT "github.com/yunba/mqtt.go"
)

//////////////////////////////////////////////////////////////////////////////
// data & struct
type ConfigJson struct {
    Ticket_host  string
    Ticket_port  int

    App_key      string
    Reg_host     string
    Reg_port     int

    Hosts_communication  string
}

type Broker struct {
    url   string
    host  string
    port  int
}

type Message struct {
    P, C, U string
}

// config json demo
// {
//     "appkey": "52fcc04c4dc903d66d6f8f92",
//     "reg_host": "reg.yunba.io",

//     "ticket_host": "tick.yunba.io",
//     "ticket_port": 9999,
//     "hosts_communication":"[\"119.9.78.46\",\"119.9.78.88\",\"119.9.77.71\"]"
// }

//////////////////////////////////////////////////////////////////////////////
// global vars define
var config_filename = "./config.json"
var bigcontent_filename = "./bigcontent"
var PLATFORM_JAVASCRIPT = 2

var config ConfigJson
var broker Broker
var username, password, clientid string
var username2, password2, clientid2 string
var topic  string
//////////////////////////////////////////////////////////////////////////////
// private func
func checkTicket() bool {
    if config.Ticket_host == "" || config.Ticket_port == 0 {
        return false;
    }

    return true;
}

func checkReg() bool {
    if config.Reg_host == "" || config.Reg_port == 0 || config.App_key == "" {
        return false;
    }

    return true;
}

func checkBroker() bool {
    if broker.url == "" || username == "" || password == "" || clientid == "" {
        return false;
    }

    return true;
}

func loadConfig() bool {
    bytes, err := ioutil.ReadFile(config_filename)
    if err != nil {
        fmt.Println("ReadFile: ", err.Error())
        return false
    }

    fmt.Println("file:", string(bytes))

    //json str 转struct
    if err := json.Unmarshal(bytes, &config); err != nil {
        fmt.Println("fail to parser config json:", err)
        return false;
    }

    //fmt.Println("================json str 转struct==")
    fmt.Println(config)
    return true;
}

func paserBroker(broker string)(ip string, port int) {
    host := strings.TrimPrefix(broker, "tcp://")
    str := strings.Split(host, ":")
    ip = str[0]
    port, err := strconv.Atoi(str[1])
    if (err != nil) {
        fmt.Println("failt to conv port:", err)
        port = 80
    }

    return ip, port  
}

func reg(appkey string) ([]byte, error) {
    url := "http://" + config.Reg_host + ":" + strconv.Itoa(config.Reg_port) + "/device/reg/"
    str := `{"a":"` + appkey + `", "p":` + strconv.Itoa(PLATFORM_JAVASCRIPT) + "}";
    buf := strings.NewReader(str)
    fmt.Println("request: ", url)
    fmt.Println("post data: ", str)

    resp, err := http.Post(url, "application/json", buf) 
    if err != nil {
        // handle error
        fmt.Println("request: ", url)
        fmt.Println("error: ", err.Error())
        return nil, err
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    fmt.Println("response:", string(body))
    return body, nil
}

func wait(c chan bool) bool {
    fmt.Println("choke is waiting")
    select {
    case  <-c:
        return true;
    case <-time.After(time.Second * 10):
        fmt.Println("wait 10s timeout")
        return false
    }
}

//////////////////////////////////////////////////////////////////////////////
// mqtt client test func

// 'should get tickect'
func Test_GetTicket(t *testing.T) {
    ret := loadConfig()
    if !ret {
        t.Fatalf("fail to load config json")
    }

    if !checkTicket() {
        t.Fatalf("fail to ready")
    }

    url := "http://" + config.Ticket_host + ":" + strconv.Itoa(config.Ticket_port) + "/"
    buf := strings.NewReader(`{"c": 'The Reddest'}`)

    fmt.Println("request: ", url)
    resp, err := http.Post(url, "application/json", buf) 
    if err != nil {
        // handle error
        // fmt.Println("request: ", url)
        fmt.Println("error: ", err.Error())
        t.Fatalf("fail to post ticket")
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    fmt.Println("response:", string(body))

    //先创建一个目标类型的实例对象，用于存放解码后的值
    var inter interface{}
    err = json.Unmarshal(body, &inter)
    if err != nil {
        fmt.Println("error in translating,", err.Error())
        t.Fatalf("fail to paser response")
    }

    // 要访问解码后的数据结构，需要先判断目标结构是否为预期的数据类型
    // 然后通过for循环一一访问解码后的目标数据
    book, ok := inter.(map[string]interface{})
    if !ok {
        t.Fatalf("fail to paser response")
    }

    for k, v := range book {
        if k != "c" {
            fmt.Println("illegle key")
            t.Fatalf("fail to paser response")
        } 

        switch vt:= v.(type) {
        case string:
            broker.url = vt
            broker.host, broker.port = paserBroker(vt)
            fmt.Println("The test broker:", broker.url)
        default:
            fmt.Println("illegle type")
            t.Fatalf("fail to paser response")
        }
    }
}

// 'should return username/password/clientid'
func Test_Reg(t *testing.T) {
    if !checkReg() {
        t.Fatalf("fail to ready")
    }

    body, err := reg(config.App_key);
    if err != nil {
        t.Fatalf("fail to reg")
    }

    dec := json.NewDecoder(strings.NewReader(string(body)))
    for {
        var m Message
        if err := dec.Decode(&m); err == io.EOF {
            break
        } else if err != nil {
            t.Fatalf("fail to reg: %s\n", err)
        }

        if (m.U == "") {
            t.Fatalf("fail to reg")
        }

        username = m.U;
        password = m.P;
        clientid = m.C
        fmt.Printf("Reg OK, user:%s, passwd:%s, clientid:%s\n", username, password, clientid)
    }
}

// 'should connect'
func Test_Connect(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    client.Disconnect(250)
}

//'should subscribe a random topic: '
func Test_Subscribe(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)

    random_topic := UUID.New();
    fmt.Println("random topic:", random_topic)
    filter, e := MQTT.NewTopicFilter(random_topic, byte(0))
    if e != nil {
        t.Fatalf("fail to fileter: %s\n", err)
    }

    _, err = client.StartSubscription(nil, filter)
    if err != nil {
        t.Fatalf("fail to StartSubscription: %s\n", err)
    }

    // for {
    //     time.Sleep(1 * time.Second)
    // }
}

//'should subscribe multiple topics by one request'
func Test_MultiSubscribe(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)

    filter1, e := MQTT.NewTopicFilter("multitopics1", byte(0))
    if e != nil {
        t.Fatalf("fail to fileter1: %s\n", err)
    }

    filter2, e := MQTT.NewTopicFilter("multitopics2", byte(0))
    if e != nil {
        t.Fatalf("fail to fileter2: %s\n", err)
    }

    filter3, e := MQTT.NewTopicFilter("multitopics3", byte(0))
    if e != nil {
        t.Fatalf("fail to fileter3: %s\n", err)
    }

    _, err = client.StartSubscription(nil, filter1, filter2, filter3)
    if err != nil {
        t.Fatalf("fail to StartSubscription: %s\n", err)
    }

    // for {
    //     time.Sleep(1 * time.Second)
    // }
}

// 'should get publish 5 messages from topic: '
var count_expected = 3;
var count_recv = 0;

func Test_MultiPublish(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)

    message1 := "test message";
    topic = UUID.New();
    fmt.Println("new topic:", topic)
    filter, e := MQTT.NewTopicFilter(topic, byte(0))
    if e != nil {
        t.Fatalf("fail to fileter: %s\n", err)
    }

    choke := make(chan bool)
    var onMessageReceived MQTT.MessageHandler = func (client *MQTT.MqttClient, message MQTT.Message) {
        fmt.Printf("Received message on topic: %s\n", message.Topic())
        fmt.Printf("Received message: %s\n", message.Payload())

        count_recv++;
        if (count_recv >= count_expected) {
            // done
            client.Disconnect(250);
            choke <- true
        }
    }

    _, err = client.StartSubscription(onMessageReceived, filter)
    if err != nil {
        t.Fatalf("fail to StartSubscription: %s\n", err)
    }

    // start new client to publish
    body, err := reg(config.App_key);
    if err != nil {
        t.Fatalf("fail to reg")
    }

    dec := json.NewDecoder(strings.NewReader(string(body)))
    for {
        var m Message
        if err := dec.Decode(&m); err == io.EOF {
            break
        } else if err != nil {
            t.Fatalf("fail to reg: %s\n", err)
        }

        if (m.U == "") {
            t.Fatalf("fail to reg")
        }

        username2 = m.U;
        password2 = m.P;
        clientid2 = m.C
        fmt.Printf("Reg OK, user:%s, passwd:%s, clientid:%s\n", username2, password2, clientid2)

        opts2 := MQTT.NewClientOptions()
        opts2.AddBroker(broker.url)
        opts2.SetClientId(clientid2)
        opts2.SetUsername(username2)
        opts2.SetPassword(password2)
        opts2.SetProtocolVersion(0x13)

        fmt.Println("Connecting to ", broker.url)
        client2 := MQTT.NewClient(opts2)
        _, err := client2.Start()
        if err != nil {
            t.Fatalf("fail to connect: %s\n", err)
        }

        defer client2.Disconnect(250)

        for i:=0; i<count_expected; i++ {
            r := client2.Publish(MQTT.QoS(byte(1)), topic, []byte(message1))
            fmt.Println(topic, " puback:", <-r) // received puback will send message to chan r,   net.go: case PUBACK
            //fmt.Println("Message Sent!")
        }

        if !wait(choke) {
            t.Fatalf("fail to receive published message for topic: %s\n", topic)
        }
    }
}

// 'should unsubscribe other topic, and DO receive message'
func Test_Unsubscribe(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    // receiver 
    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    // set default handler
    choke := make(chan bool)
    message1 := "test message";
    opts.SetDefaultPublishHandler(func(client *MQTT.MqttClient, msg MQTT.Message) {
        //check 
        fmt.Printf("Received message on topic: %s\n", msg.Topic())
        fmt.Printf("Received message: %s\n", msg.Payload())
        if msg.Topic() == topic && string(msg.Payload()) == message1 {
            client.Disconnect(250)
            choke <- true
        } else {
            t.Fatalf("mistake memssage!")
        }
    })

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)

    _, err = client.EndSubscription(topic+"1")
    if err != nil {
        t.Fatalf("fail to EndSubscription: %s\n", err)
    }

    // sender
    opts2 := MQTT.NewClientOptions()
    opts2.AddBroker(broker.url)
    opts2.SetClientId(clientid2)
    opts2.SetUsername(username2)
    opts2.SetPassword(password2)
    opts2.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client2 := MQTT.NewClient(opts2)
    _, err = client2.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client2.Disconnect(250)
    r := client2.Publish(MQTT.QoS(byte(1)), topic, []byte(message1))
    fmt.Println("puback:", <-r) // received puback will send message to chan r,   net.go: case PUBACK
    if !wait(choke) {
        t.Fatalf("fail to receive published message for topic: %s\n", topic)
    }
}

// 'should get a puback when publish to a new topic'
func Test_PublishNew(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    random_topic := UUID.New()
    r := client.Publish(MQTT.QoS(byte(1)), random_topic, []byte("test"))
    fmt.Println("Message Sent to new topic:", random_topic)
    fmt.Println("puback:", <-r) // received puback will send message to chan r,   net.go: case PUBACK
    client.Disconnect(250)
}

// 'should get a puback when unsubscribed topic has no uids'
func Test_PublishUnsubed(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)
    random_topic := UUID.New()
    test_message := "hello";
    fmt.Println("random topic:", random_topic)
    filter, e := MQTT.NewTopicFilter(random_topic, byte(1))
    if e != nil {
        t.Fatalf("fail to fileter: %s\n", err)
    }

    choke := make(chan bool)
    var onMessageReceived MQTT.MessageHandler = func (client *MQTT.MqttClient, message MQTT.Message) {
        fmt.Printf("Received message on topic: %s\n", message.Topic())
        fmt.Printf("Received message: %s\n", message.Payload())
        if message.Topic() == random_topic && string(message.Payload()) == test_message {
            _, err = client.EndSubscription(random_topic)
            if err != nil {
                t.Fatalf("fail to EndSubscription: %s\n", err)
            }

            r1 := client.Publish(MQTT.QoS(byte(1)), random_topic, []byte(test_message))
            fmt.Println("r1 puback:", <-r1) // received puback will send message to chan r,   net.go: case PUBACK

            // done
            choke <- true
        }
    }

    _, err = client.StartSubscription(onMessageReceived, filter)
    if err != nil {
        t.Fatalf("fail to StartSubscription: %s\n", err)
    }

    r := client.Publish(MQTT.QoS(byte(1)), random_topic, []byte(test_message))
    fmt.Println("puback: ", <-r) // received puback will send message to chan r,   net.go: case PUBACK

    if !wait(choke) {
        t.Fatalf("fail to receive published message for topic: %s\n", random_topic)
    }
}

// 'should get a message when publish large content'
func Test_PublishLarge(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)
    random_topic := UUID.New()
    fmt.Println("random topic:", random_topic)
    filter, e := MQTT.NewTopicFilter(random_topic, byte(0))
    if e != nil {
        t.Fatalf("fail to fileter: %s\n", err)
    }

    // read bigcontent from file
    bigcontent, err := ioutil.ReadFile(bigcontent_filename)
    if err != nil {
        t.Fatalf("fail to ReadFile: %s\n", err.Error())
    }

    choke := make(chan bool)
    var onMessageReceived MQTT.MessageHandler = func (client *MQTT.MqttClient, message MQTT.Message) {
        fmt.Printf("Received message on topic: %s\n", message.Topic())
        //fmt.Printf("Received message: %s\n", message.Payload())
        if message.Topic() == random_topic && string(message.Payload()) == string(bigcontent) {
            // done
            choke <- true
        }
    }

    _, err = client.StartSubscription(onMessageReceived, filter)
    if err != nil {
        t.Fatalf("fail to StartSubscription: %s\n", err)
    }

    r := client.Publish(MQTT.QoS(byte(1)), random_topic, bigcontent)
    fmt.Println("puback:", <-r) // received puback will send message to chan r,   net.go: case PUBACK

    if !wait(choke) {
        t.Fatalf("fail to receive published message for topic: %s\n", random_topic)
    }
}

// func except() {
//     recover()
// }

// 'should unsubscribe the topic, and DO NOT receive message: 
func Test_UnsubscribeTopic(t *testing.T) {
    if !checkBroker() {
        t.Fatalf("fail to ready")
    }

    // receiver 
    opts := MQTT.NewClientOptions()
    opts.AddBroker(broker.url)
    opts.SetClientId(clientid)
    opts.SetUsername(username)
    opts.SetPassword(password)
    opts.SetProtocolVersion(0x13)

    // set default handler
    message1 := "test message";
    opts.SetDefaultPublishHandler(func(client *MQTT.MqttClient, msg MQTT.Message) {
        //check 
        fmt.Printf("Received message on topic: %s\n", msg.Topic())
        fmt.Printf("Received message: %s\n", msg.Payload())
        if msg.Topic() == topic {
            t.Fatalf("I should not receive message anymore, after unsubscribe topic = %s receiver msg from %s", topic, msg.Topic())
        }
    })

    fmt.Println("Connecting to ", broker.url)
    client := MQTT.NewClient(opts)
    _, err := client.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    defer client.Disconnect(250)

    _, err = client.EndSubscription(topic)
    if err != nil {
        t.Fatalf("fail to EndSubscription: %s\n", err)
    }

    // sender
    opts2 := MQTT.NewClientOptions()
    opts2.AddBroker(broker.url)
    opts2.SetClientId(clientid2)
    opts2.SetUsername(username2)
    opts2.SetPassword(password2)
    opts2.SetProtocolVersion(0x13)

    fmt.Println("Connecting to ", broker.url)
    client2 := MQTT.NewClient(opts2)
    _, err = client2.Start()
    if err != nil {
        t.Fatalf("fail to connect: %s\n", err)
    }

    // defer except()
    // panic("test panic")

    r := client2.Publish(MQTT.QoS(byte(1)), topic, []byte(message1))
    fmt.Println("puback:", <-r) // received puback will send message to chan r,   net.go: case PUBACK
    time.Sleep(3 * time.Second)
    client2.Disconnect(250)
}

// 'should get ticket1 success'






