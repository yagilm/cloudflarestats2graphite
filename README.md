# cloudflarestats2graphite
Pulls zone analytics from CloudFlare API and pushes them to a graphite

## What does it do
The program uses cloudflare's API to pull zone information and sends the Status Codes to a graphite.
It requires that you have an API key as described here: https://api.cloudflare.com/#getting-started-endpoints

## How to use
All options are required:
```
Usage: cloudflareanalytics [options]
Required options:
  --auth string
        X-Auth-Key for cloudflare's API
  --email string
        X-Auth-Email for cloudflare's API
  --ghost string
        Graphite host
  --gport int
        Graphite port
  --zone string
        Cloudflare's zone
  --zonedomain string
        Domain of the zone
```
