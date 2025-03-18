package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	flag "github.com/spf13/pflag"
	"golang.org/x/net/publicsuffix"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

const (
	appName = "cfddns"
	timeout = 10 * time.Second
)

var (
	version    string
	record     string
	token      string
	domain     string
	ip         string
	verbose    bool
	force      bool
	httpClient = &http.Client{Timeout: timeout}
)

func validateIPv4(ip string) error {
	if parsed := net.ParseIP(ip); parsed == nil || parsed.To4() == nil {
		return fmt.Errorf("invalid IP")
	}
	return nil
}

func getIPv4() (string, error) {
	sources := []string{
		"https://ipv4.icanhazip.com",
		"https://ipv4.ifconfig.me",
		"https://api.ipify.org",
	}

	for _, url := range sources {
		resp, err := httpClient.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}
		ipBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		ip := strings.TrimSpace(string(ipBytes))
		if err := validateIPv4(ip); err == nil {
			return ip, nil
		}
	}
	return "", fmt.Errorf("failed to get IPv4 address")
}

func getZone(client *cloudflare.Client, domain string) (*zones.Zone, error) {
	zone, err := client.Zones.List(context.Background(), zones.ZoneListParams{Name: cloudflare.F(domain)})
	if err != nil {
		return &zones.Zone{}, err
	}
	if len(zone.Result) < 1 {
		return &zones.Zone{}, fmt.Errorf("zone not found")
	}
	return &zone.Result[0], nil
}

func getRecord(client *cloudflare.Client, zoneID, name string) (*dns.RecordResponse, error) {
	record, err := client.DNS.Records.List(context.Background(), dns.RecordListParams{
		ZoneID: cloudflare.F(zoneID),
		Name:   cloudflare.F(dns.RecordListParamsName{Exact: cloudflare.F(name)}),
		Type:   cloudflare.F(dns.RecordListParamsTypeA),
	})
	if err != nil {
		return &dns.RecordResponse{}, err
	}
	if len(record.Result) < 1 {
		return &dns.RecordResponse{}, fmt.Errorf("record not found")
	}
	return &record.Result[0], nil
}

func updateRecord(client *cloudflare.Client, zoneID, recordID, ip string) (*dns.RecordResponse, error) {
	if ip == "" {
		panic("empty IP")
	}

	record, err := client.DNS.Records.Edit(context.Background(), recordID, dns.RecordEditParams{
		ZoneID: cloudflare.F(zoneID),
		Record: dns.ARecordParam{Content: cloudflare.F(ip)},
	})
	if err != nil {
		return &dns.RecordResponse{}, err
	}
	return record, nil
}

func validateArgs() error {
	if record == "" {
		return fmt.Errorf("Error: missing record")
	}
	if token == "" {
		if x := os.Getenv("CLOUDFLARE_API_TOKEN"); x == "" {
			return fmt.Errorf("Error: missing token")
		} else {
			token = x
		}
	}
	if domain == "" {
		if x, err := publicsuffix.EffectiveTLDPlusOne(record); err != nil {
			return fmt.Errorf("Error: could not determine zone domain, use --domain")
		} else {
			domain = x
		}
	}
	if ip == "" {
		if x, err := getIPv4(); err != nil {
			return fmt.Errorf("Error: %w", err)
		} else {
			ip = x
		}
	} else {
		if err := validateIPv4(ip); err != nil {
			return fmt.Errorf("Error: invalid IP")
		}
	}
	if record == domain && !force {
		return fmt.Errorf("Error: --force required to update root domain")
	}
	return nil
}

func errExit(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func init() {
	flag.StringVarP(&token, "token", "t", "", "Cloudflare API token [CLOUDFLARE_API_TOKEN]")
	flag.StringVarP(&domain, "domain", "d", "", "zone name (default record domain)")
	flag.StringVar(&ip, "ip", "", "IP address (default automatically resolved)")
	flag.BoolVarP(&force, "force", "f", false, "force update (required only for root domain)")
	flag.BoolVarP(&verbose, "verbose", "v", false, "verbose")
	flag.BoolP("help", "h", false, "display usage help")
	flag.BoolP("version", "V", false, "display version")
	flag.CommandLine.SortFlags = false
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Cloudflare DDNS client\n\n")
		fmt.Fprintf(os.Stderr, "Usage: "+appName+" [options...] <record>\n\n")
		flag.PrintDefaults()
	}
	if version == "" {
		version = time.Now().Format("2006.1.2-dev")
	}
}

func main() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			flag.Usage()
			os.Exit(0)
		case "--version", "-V":
			fmt.Printf("%s v%s %s/%s\n", appName, version, runtime.GOOS, runtime.GOARCH)
			fmt.Printf("Copyright (C) %s Derek Nicol. Licensed under GNU GPLv3. Not affiliated with Cloudflare\n", strings.Split(version, ".")[0])
			os.Exit(0)
		}
	}
	flag.Parse()
	record = flag.Arg(0)
	if err := validateArgs(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cf := cloudflare.NewClient(option.WithAPIToken(token), option.WithHTTPClient(httpClient))

	zone, err := getZone(cf, domain)
	if err != nil {
		errExit("Error: %v", err)
	}
	record, err := getRecord(cf, zone.ID, record)
	if err != nil {
		errExit("Error: %v", err)
	}
	if record.Content == ip {
		fmt.Println("No update needed")
		if verbose {
			if json, err := json.MarshalIndent(record, "", "  "); err == nil {
				fmt.Println(string(json))
			}
		}
		os.Exit(0)
	}

	newRecord, err := updateRecord(cf, zone.ID, record.ID, ip)
	if err != nil {
		errExit("Error: %v", err)
	}

	ttl := "auto"
	if newRecord.TTL > 1 {
		ttl = fmt.Sprintf("%0.0f", newRecord.TTL)
	}
	fmt.Printf("Updated %s to %s from %s (TTL %s)\n", newRecord.Name, newRecord.Content, record.Content, ttl)
	if verbose {
		if json, err := json.MarshalIndent(newRecord, "", "  "); err == nil {
			fmt.Println(string(json))
		}
	}
}
