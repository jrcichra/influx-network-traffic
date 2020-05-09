package influx

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jrcichra/influx-network-traffic/packet"

	"github.com/influxdata/influxdb/client"
)

//Connection - details to connect to influx
type Connection struct {
	Hostname string
	Db       string
	Username string
	Password string
	Port     int
}

//Influx - wrapper to influx client with state
type Influx struct {
	client     *client.Client
	Connection Connection
}

//Connect - connect to the influx database
func (f *Influx) Connect(influxConn Connection) {
	f.Connection = influxConn
	u, err := url.Parse(fmt.Sprintf("http://%s:%d", influxConn.Hostname, influxConn.Port))
	if err != nil {
		log.Fatal(err)
	}

	f.client, err = client.NewClient(client.Config{URL: *u})
	if err != nil {
		log.Fatal(err)
	}

	if _, _, err := f.client.Ping(); err != nil {
		log.Fatal(err)
	}

	f.client.SetAuth(influxConn.Username, influxConn.Password)

	log.Println("Connected to influxdb:", influxConn.Hostname, influxConn.Port, "as", influxConn.Username)
}

//Write - writes just as the script does
func (f *Influx) Write(measurement string, packet packet.Packet, interval time.Duration, t time.Time) (*client.Response, error) {

	//Convert struct to JSON so we get the keys we want
	jsonB, err := json.Marshal(packet)
	if err != nil {
		panic(err)
	}

	//Convert it into a map[string][string] for the tags
	var m map[string]string
	err = json.Unmarshal(jsonB, &m)
	if err != nil {
		panic(err)
	}

	//Remove bytes as it's not a tag but a field (the only one)
	delete(m, "bytes")

	pt := client.Point{
		Measurement: "throughput",
		Tags:        m,
		Fields: map[string]interface{}{
			"throughput": packet.Bytes,
			"interval":   int(interval.Seconds()),
		},
		Time: t,
	}

	pts := make([]client.Point, 1)
	pts = append(pts, pt)

	bps := client.BatchPoints{
		Points:          pts,
		Database:        f.Connection.Db,
		RetentionPolicy: "autogen",
	}

	// Return back the write error
	return f.client.Write(bps)
}