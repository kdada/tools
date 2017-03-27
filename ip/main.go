package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var path string

func init() {
	flag.StringVar(&path, "f", "http://ftp.apnic.net/apnic/stats/apnic/delegated-apnic-latest", "APNIC file path or a url")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.Parse()

	data, err := FetchData(path)
	if err != nil {
		log.Fatal(err)
	}
	result := FindCNIPNet(data)
	log.Println(result)
}
func FetchData(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return FetchDataFromUrl(path)
	}
	return FetchDataFromFile(path)
}

func FetchDataFromUrl(path string) (string, error) {
	resp, err := http.Get(path)
	if err != nil {
		return "", err
	}
	// 200 is ok. Other codes are failed.
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("can't download file with code: %d", resp.StatusCode)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FetchDataFromFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type IPBlock struct {
	IP   uint32
	Mask byte
}

func (b *IPBlock) String() string {
	ip := ConvertIntToIP(b.IP)
	return fmt.Sprintf("%s/%d", ip.String(), b.Mask)
}

// ConvertIntToIP converts a value to IP
func ConvertIntToIP(v uint32) net.IP {
	ip := make(net.IP, 4)
	for i := 0; i < 4 && v > 0; i++ {
		ip[3-i] = byte(v)
		v >>= 8
	}
	return ip
}

// ConvertIPToInt converts ip to int value
func ConvertIPToInt(ip net.IP) uint32 {
	value := uint32(0)
	for i := 0; i < 4; i++ {
		value = value<<8 | uint32(ip[i])
	}
	return value
}

var cnReg = regexp.MustCompile(`apnic\|CN\|ipv4\|(\d+\.\d+\.\d+\.\d+)\|(\d+)\|`)

func FindCNIPNet(data string) []*IPBlock {
	result := cnReg.FindAllStringSubmatch(data, -1)
	ips := make([]*IPBlock, len(result))
	for i, r := range result {
		count, _ := strconv.Atoi(r[2])
		mask := byte(32 - math.Log2(float64(count)))

		ips[i] = &IPBlock{
			IP:   ConvertIPToInt(net.ParseIP(r[1]).To4()),
			Mask: mask,
		}
	}
	return ips
}

func Reverse(origin []*IPBlock) []*IPBlock {
	return nil
}

func Merge(origin []*IPBlock) []*IPBlock {
	return nil
}
