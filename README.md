# Cloudflare Dynamic DNS Client

![GitHub Release](https://img.shields.io/github/v/release/derekn/cfddns)
![GitHub License](https://img.shields.io/github/license/derekn/cfddns)

An unofficial Cloudflare client for updating DNS records.

## Installation

Download from Github [releases](https://github.com/derekn/cfddns/releases/latest).  
or, install using Go:

```shell
go install https://github.com/derekn/cfddns@latest
```

## Usage

```shell
# use API token from environment variable (recommended)
export CLOUDFLARE_API_TOKEN=xxxx
cfddns record.domain.tld

# pass token as argument
cfddns --token xxxx record.domain.tld
```

### Arguments

```shell
-t, --token string    Cloudflare API token [CLOUDFLARE_API_TOKEN]
-d, --domain string   zone name (default record domain)
    --ip string       IP address (default automatically resolved)
-v, --verbose         verbose
-h, --help            display usage help
-V, --version         display version
```
