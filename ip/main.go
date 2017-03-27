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
	"sort"
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
	result = AddReservedBlock(result)
	result = Reverse(result)
	sort.Slice(result, func(i, j int) bool {
		return result[i].IP < result[j].IP
	})
	log.Println(result)
	log.Println(len(result))
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

func AddReservedBlock(origin []*IPBlock) []*IPBlock {
	ips := []string{"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16",
		"172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16",
		"198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4"}
	b := make([]*IPBlock, len(origin)+len(ips))
	for i, ip := range ips {
		_, r, _ := net.ParseCIDR(ip)
		m, _ := r.Mask.Size()
		b[i] = &IPBlock{
			IP:   ConvertIPToInt(r.IP),
			Mask: byte(m),
		}
	}
	copy(b[len(ips):], origin)
	return b
}

type Bit struct {
	// 0
	Zero *Bit
	// 1
	One *Bit
}

func Reverse(origin []*IPBlock) []*IPBlock {
	root := &Bit{}
	for _, o := range origin {
		Generate(root, o)
	}
	result := make([]*IPBlock, 0, len(origin))
	Merge(&result, root, 0, 0)
	return result
}

func Generate(root *Bit, b *IPBlock) {
	current := root
	mask := uint32(0x80000000)
	for i := 0; i < int(b.Mask); i++ {
		if b.IP&mask > 0 {
			if current.One == nil {
				current.One = &Bit{}
			}
			current = current.One
		} else {
			if current.Zero == nil {
				current.Zero = &Bit{}
			}
			current = current.Zero
		}
		mask >>= 1
	}
}

func Merge(result *[]*IPBlock, current *Bit, parent uint32, level uint) {
	if current.One == nil && current.Zero == nil {
		return
	}
	if current.One != nil {
		Merge(result, current.One, parent+1<<(31-level), level+1)
	}
	if current.Zero != nil {
		Merge(result, current.Zero, parent, level+1)
	}
	if current.One != nil && current.Zero != nil {
		return
	}
	ip := parent
	mask := level + 1
	if current.One == nil {
		ip += 1 << (31 - level)
	}
	*result = append(*result, &IPBlock{
		IP:   ip,
		Mask: byte(mask),
	})
}
