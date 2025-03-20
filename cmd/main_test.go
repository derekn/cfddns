package main

import (
	"os"
	"testing"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/joho/godotenv"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/publicsuffix"
)

func TestValidateIPv4(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"Valid", "192.168.1.1", false},
		{"Invalid", "256.256.256.256", true},
		{"IPv6", "2001:db8::ff00:42", true},
		{"String", "invalid", true},
		{"Empty", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateIPv4(test.ip); (err != nil) != test.wantErr {
				t.Errorf("expected %v got %v", test.wantErr, (err != nil))
			}
		})
	}
}

func TestValidation(t *testing.T) {
	if err := godotenv.Load("../test.env"); err != nil {
		panic(err)
	}
	testIP := os.Getenv("TEST_IP")

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"Empty", []string{"app"}, true},
		{"NoToken", []string{"app", "foo.bar.tld"}, true},
		{"MinimalFlags", []string{"app", "foo.bar.tld"}, false},
		{"TokenFromFlag", []string{"app", "foo.bar.tld", "-t", "def456"}, false},
		{"CustomDomain", []string{"app", "foo.bar.tld", "--domain", "test.tld"}, false},
		{"RootDomain", []string{"app", "bar.tld"}, true},
		{"RootDomainForce", []string{"app", "bar.tld", "--force"}, false},
		{"CustomIP", []string{"app", "foo.bar.tld", "--ip", "192.168.0.1"}, false},
		{"InvalidCustomIP", []string{"app", "foo.bar.tld", "--ip", "192.168.0.256"}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record, token, domain, ip = "", "", "", ""
			os.Setenv("CLOUDFLARE_API_TOKEN", "abc123")
			if test.name == "NoToken" {
				os.Unsetenv("CLOUDFLARE_API_TOKEN")
			}

			os.Args = test.args
			flag.Parse()
			record = flag.Arg(0)
			if err := validateArgs(); (err != nil) != test.wantErr {
				t.Errorf("expected %v got %v - %v", test.wantErr, (err != nil), err)
			}

			if test.name == "MinimalFlags" {
				if domain != "bar.tld" {
					t.Errorf("expected \"bar.tld\" got %q", domain)
				}
				if ip != testIP {
					t.Errorf("expected %q got %q", testIP, ip)
				}
			}
			if test.name == "TokenFromFlag" && token != test.args[3] {
				t.Errorf("expected %q got %q", test.args[3], domain)
			}
			if test.name == "CustomDomain" && domain != test.args[3] {
				t.Errorf("expected %q got %q", test.args[3], domain)
			}
			if test.name == "CustomIP" && ip != test.args[3] {
				t.Errorf("expected %q got %q", test.args[3], ip)
			}
		})
	}
}

func TestMain(t *testing.T) {
	if err := godotenv.Overload("../test.env"); err != nil {
		panic(err)
	}

	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	testRecord := os.Getenv("TEST_RECORD")
	testZone := os.Getenv("TEST_ZONE")
	testIP := os.Getenv("TEST_IP")

	domain, err := publicsuffix.EffectiveTLDPlusOne(testRecord)
	if err != nil {
		panic(err)
	}

	var (
		ip   string
		zone *zones.Zone
		// record *dns.RecordResponse
		ok bool
	)

	ok = t.Run("GetIPv4", func(t *testing.T) {
		x, err := getIPv4()
		if err != nil {
			t.Fatal(err.Error())
		}
		if x != testIP {
			t.Fatalf("expected %q got %q", testIP, ip)
		}
		ip = x
	})
	if !ok {
		return
	}

	cf := cloudflare.NewClient(option.WithAPIToken(token))

	ok = t.Run("GetZone", func(t *testing.T) {
		x, err := getZone(cf, domain)
		if err != nil {
			t.Fatal(err.Error())
		}
		if x.ID != testZone {
			t.Fatalf("expected %q got %q", testZone, x.ID)
		}
		zone = x
	})
	if !ok {
		return
	}

	t.Run("GetRecord", func(t *testing.T) {
		x, err := getRecord(cf, zone.ID, testRecord)
		if err != nil {
			t.Fatal(err.Error())
		}
		if x.Name != testRecord {
			t.Fatalf("expected %q got %q", testRecord, x.Name)
		}
		// record = x
	})
}
