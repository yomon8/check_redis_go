package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Redis Protcol
const (
	ErrorReply  = '-'
	StatusReply = '+'
	IntReply    = ':'
	StringReply = '$'
	ArrayReply  = '*'
)

// RedisConn is connection paramaters for redis
type client struct {
	conn     net.Conn
	host     string
	port     int
	password string
}

// NagiosReturn handle plugin return codes
type NagiosReturn struct {
	currentCode int
}

func (nr *NagiosReturn) setReturnCode(code int) {
	if code > nr.currentCode {
		nr.currentCode = code
	}
}

// Nagios return codes
const (
	NagiosOk       = 0
	NagiosWarning  = 1
	NagiosCritical = 2
	NagiosUnknown  = 3
)

type threshold struct {
	isset bool
	value float64
}

func (t *threshold) setValue(f float64) {
	t.isset = true
	t.value = f
}

func newClient(host string, port int, password string, timeoutmsec int) (*client, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Duration(timeoutmsec)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	c := &client{conn: conn, host: host, port: port, password: password}
	return c, nil
}

func (c *client) info() (string, error) {
	defer c.conn.Close()
	var (
		wb []byte
		rb = make([]byte, 4096)
	)

	reader := bufio.NewReader(c.conn)
	if c.password != "" {
		cmd := fmt.Sprintf("%s \"%s\"", "AUTH", c.password)
		wb = append(wb, cmd...)
		wb = append(wb, '\r', '\n')
		c.conn.Write(wb)
		authReply, err := reader.ReadBytes('\n')
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(string(authReply), string(ErrorReply)) {
			return "", fmt.Errorf("Authorization failed:%s", string(authReply))
		}
	}
	wb = nil
	wb = append(wb, "INFO"...)
	wb = append(wb, '\r', '\n')
	c.conn.Write(wb)
	stringLength, _, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	var size int
	if strings.HasPrefix(string(stringLength), string(ErrorReply)) {
		return "", fmt.Errorf("Authorization failed:%s", string(stringLength))
	} else if strings.HasPrefix(string(stringLength), string(StringReply)) {
		size, err = strconv.Atoi(string(stringLength)[1:])
		if err != nil {
			return "", err
		}
	}

	if cap(rb) >= size {
		rb = rb[:size]
		io.ReadFull(reader, rb)
	} else {
		pos := 0
		bytesAllocLimit := 1024 * 1024 // 1MB
		for pos < size {
			diff := size - len(rb)
			if diff > bytesAllocLimit {
				diff = bytesAllocLimit
			}
			rb = append(rb, make([]byte, diff)...)
			nn, err := io.ReadFull(reader, rb[pos:])
			if err != nil {
				return "", err
			}
			pos += nn
		}
	}
	return string(rb), nil
}

func fmtResult(severity, name, op string, value float64, tw, tc threshold) string {
	res := fmt.Sprintf("[%s]%s:%.1f %s ", severity, name, value, op)
	switch {
	case tw.isset && tc.isset:
		res += fmt.Sprintf("(warn%.1f crit%.1f)\n", tw.value, tc.value)
	case tw.isset:
		res += fmt.Sprintf("(warn%.1f)\n", tw.value)
	case tc.isset:
		res += fmt.Sprintf("(crit%.1f)\n", tc.value)
	default:
		res += fmt.Sprintf("no threshold\n")
	}
	return res
}

func parseMetrics(metrics string, redisresult *string, nr *NagiosReturn) (result string) {
	for _, metricString := range strings.Split(metrics, ",") {
		var (
			metricName        string
			tc, tw            float64
			thresholdWarning  threshold
			thresholdCritical threshold
			op                = "gt"
			parseErr          error
		)
		for i, v := range strings.Split(metricString, ":") {
			switch i {
			case 0:
				metricName = v
			case 1:
				if v != "" {
					tw, parseErr = strconv.ParseFloat(v, 64)
					thresholdWarning.setValue(tw)
				}
			case 2:
				if v != "" {
					tc, parseErr = strconv.ParseFloat(v, 64)
					thresholdCritical.setValue(tc)
				}
			case 3:
				if v == "lt" {
					op = v
				}
			}
		}

		if parseErr != nil || metricName == "" {
			result += fmt.Sprintf("[Unknown]metrics parameter invalid (%s)\n", metricString)
			nr.setReturnCode(NagiosUnknown)
			continue
		}
		var existLine bool
		for _, line := range strings.Split(*redisresult, "\n") {
			if strings.HasPrefix(line, fmt.Sprintf("%s:", metricName)) {
				existLine = true
				val := strings.TrimSpace(strings.Split(line, ":")[1])
				metricValue, err := strconv.ParseFloat(val, 64)
				switch {
				case err != nil:
					nr.setReturnCode(NagiosUnknown)
					fmt.Printf("[Unknown]%s\n", metricName)
				case thresholdCritical.isset && op == "gt" && metricValue > thresholdCritical.value:
					nr.setReturnCode(NagiosCritical)
					result += fmtResult(
						"CRIT",
						metricName,
						op,
						metricValue,
						thresholdWarning,
						thresholdCritical)
				case thresholdCritical.isset && op == "lt" && metricValue < thresholdCritical.value:
					nr.setReturnCode(NagiosCritical)
					result += fmtResult(
						"CRIT",
						metricName,
						op,
						metricValue,
						thresholdWarning,
						thresholdCritical)
				case thresholdWarning.isset && op == "gt" && metricValue > thresholdWarning.value:
					nr.setReturnCode(NagiosWarning)
					result += fmtResult(
						"WARN",
						metricName,
						op,
						metricValue,
						thresholdWarning,
						thresholdCritical)
				case thresholdWarning.isset && op == "lt" && metricValue < thresholdWarning.value:
					nr.setReturnCode(NagiosWarning)
					result += fmtResult(
						"WARN",
						metricName,
						op,
						metricValue,
						thresholdWarning,
						thresholdCritical)
				default:
					result += fmtResult(
						"OK",
						metricName,
						op,
						metricValue,
						thresholdWarning,
						thresholdCritical)
				}
			}
		}
		if existLine != true {
			nr.setReturnCode(NagiosUnknown)
			result += fmt.Sprintf("[Unknown]metrics %s\n", metricName)
		}
	}
	return result
}

func main() {
	var (
		host        string
		port        int
		password    string
		timeoutmsec int
		metrics     string
		nr          NagiosReturn
	)
	const (
		defaultHost    = "127.0.0.1"
		defaultPort    = 6379
		defaultTimeout = 1000 // 1sec
	)

	flag.IntVar(&port, "p", defaultPort, "port to connect redis")
	flag.IntVar(&port, "port", defaultPort, "port to connect redis")
	flag.IntVar(&timeoutmsec, "timeout", defaultTimeout, "connection timeout(msec)")
	flag.StringVar(&host, "h", defaultHost, "redis server host")
	flag.StringVar(&host, "host", defaultHost, "redis server host")
	flag.StringVar(&password, "password", "", "redis server password")
	flag.StringVar(&metrics, "metrics", "", "metrics and threshold to be monitored \"metricname:warnstring:criticalstring,...\" (e.g. used_memory:1073741824:2147483648:gt,metric2:10:20:lt..)")
	flag.Parse()

	c, err := newClient(host, port, password, timeoutmsec)
	if err != nil {
		//Catch connection error
		fmt.Printf("[Unknown]%s\n", err.Error())
		nr.setReturnCode(NagiosUnknown)
		os.Exit(nr.currentCode)
	}

	info, err := c.info()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(NagiosUnknown)
	}
	fmt.Println("[OK]Connection and Authorization Ok")
	if metrics != "" {
		res := parseMetrics(metrics, &info, &nr)
		fmt.Print(res)
	}

	os.Exit(nr.currentCode)
}
