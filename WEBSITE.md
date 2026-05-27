# Hosting AI Radio FM Publicly

This document covers everything needed to make your station(s) publicly accessible on the internet — a domain, a server, TLS, nginx proxying, and a minimal web page that embeds the Icecast stream and shows live now-playing metadata.

---

## What you are building

A static web page served over HTTPS that:
- Embeds an HTML5 audio player pointed at your Icecast mount(s)
- Polls your station's `/now-playing` API to display what is currently playing
- Works in any modern browser without plugins

No additional backend is required beyond what is already running.

---

## Infrastructure

### 1. A server with a public IP

Your `airadio` binary, Icecast, go-music-gen, and the TTS sidecar all need to run on a machine reachable from the internet.

Recommended options:

| Provider | Instance | vCPU | RAM | Cost/mo | Notes |
|---|---|---|---|---|---|
| Hetzner | CAX11 | 2 ARM | 4 GB | ~$5 | Best value; ARM is fine for ACE-Step CPU inference |
| DigitalOcean | Basic Droplet | 2 | 4 GB | $24 | Easy setup, good docs |
| Linode/Akamai | Nanode 4GB | 2 | 4 GB | $24 | Reliable |
| AWS EC2 | t3.medium | 2 | 4 GB | ~$30 | More complex networking |

For a single station, 4 GB RAM is the practical minimum (go-music-gen's ACE-Step model uses ~2–3 GB). For multiple stations sharing one TTS sidecar and one go-music-gen instance, 8 GB is more comfortable.

### 2. A domain name

Point an A record at your server's public IP. For example:

```
radio.yourdomain.com  A  <server-ip>
```

DNS propagation typically takes a few minutes to an hour.

### 3. Firewall rules

Open the following ports inbound:

| Port | Protocol | Purpose |
|---|---|---|
| 22 | TCP | SSH |
| 80 | TCP | HTTP (redirects to HTTPS) |
| 443 | TCP | HTTPS (web page + proxied stream) |

Do **not** expose port 8000 (Icecast) or 8001+ (airadio API) directly — nginx proxies both through 443.

---

## Server setup

### Install dependencies

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y nginx certbot python3-certbot-nginx icecast2 \
    espeak-ng ffmpeg git curl

# Install Go (if not present)
curl -OL https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc

# Install ONNX Runtime (for TTS sidecar)
# Download the appropriate release from https://github.com/microsoft/onnxruntime/releases
# and place libonnxruntime.so in /usr/local/lib/
```

### Clone repositories

```bash
cd ~/repo

git clone https://github.com/chaseputnam/ai-radio-fm.git
git clone https://github.com/user/go-kokoro-tts.git
git clone https://github.com/kortexa-ai/go-music-gen.git

# Download ACE-Step model checkpoints (requires the Python music-gen.server repo)
git clone https://github.com/kortexa-ai/music-gen.server.git
cd music-gen.server && ./setup.sh
```

### Build binaries

```bash
# airadio station binary
cd ~/repo/ai-radio-fm
go build -o airadio .

# TTS sidecar
go build -o ../go-kokoro-tts/tts-server ../go-kokoro-tts/cmd/tts-server/

# go-music-gen server
cd ~/repo/go-music-gen
go build -o go-music-gen ./cmd/server/
```

---

## Icecast configuration

Edit `/etc/icecast2/icecast.xml`. Key settings to change before going public:

```xml
<icecast>
  <location>Your City</location>
  <admin>admin@yourdomain.com</admin>

  <limits>
    <clients>100</clients>
    <sources>10</sources>
  </limits>

  <authentication>
    <!-- Change all three passwords from the defaults -->
    <source-password>your-strong-source-password</source-password>
    <relay-password>your-strong-relay-password</relay-password>
    <admin-user>admin</admin-user>
    <admin-password>your-strong-admin-password</admin-password>
  </authentication>

  <listen-socket>
    <port>8000</port>
    <!-- Bind to localhost only — nginx proxies public traffic -->
    <bind-address>127.0.0.1</bind-address>
  </listen-socket>

  <http-headers>
    <header name="Access-Control-Allow-Origin" value="*" />
  </http-headers>
</icecast>
```

Restart Icecast after editing:

```bash
sudo systemctl restart icecast2
sudo systemctl enable icecast2
```

Update your station's `.env` or `stations/.env.shared` to use the new password:

```bash
ICECAST_PASSWORD=your-strong-source-password
```

---

## TLS certificate

```bash
sudo certbot --nginx -d radio.yourdomain.com
```

Certbot will automatically configure nginx for HTTPS and set up auto-renewal. Verify renewal works:

```bash
sudo certbot renew --dry-run
```

---

## nginx configuration

Create `/etc/nginx/sites-available/airadio`:

```nginx
server {
    listen 443 ssl;
    server_name radio.yourdomain.com;

    ssl_certificate     /etc/letsencrypt/live/radio.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/radio.yourdomain.com/privkey.pem;

    # Serve the static web page
    root /var/www/airadio;
    index index.html;

    # Proxy Icecast streams
    # /stream/<mount> → http://localhost:8000/<mount>
    location /stream/ {
        proxy_pass http://localhost:8000/;
        proxy_set_header Host $host;
        proxy_buffering off;       # critical — do not buffer audio streams
        proxy_cache off;
        proxy_read_timeout 3600s;  # keep long-lived listener connections open
        add_header Access-Control-Allow-Origin *;
    }

    # Proxy the airadio now-playing / health API
    # /api/ → http://localhost:8001/
    location /api/ {
        proxy_pass http://localhost:8001/;
        proxy_set_header Host $host;
        add_header Access-Control-Allow-Origin *;
    }
}

# Redirect all HTTP to HTTPS
server {
    listen 80;
    server_name radio.yourdomain.com;
    return 301 https://$host$request_uri;
}
```

Enable and reload:

```bash
sudo ln -s /etc/nginx/sites-available/airadio /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

---

## Web page

Create `/var/www/airadio/index.html`. This is a minimal single-station example — extend it for multiple stations as needed.

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>WAIR-FM</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      background: #0a0a0a;
      color: #e0e0e0;
      font-family: monospace;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      gap: 0.5rem;
    }

    h1 {
      font-size: 2.5rem;
      letter-spacing: 0.3em;
      text-transform: uppercase;
    }

    #tagline {
      font-size: 0.75rem;
      color: #555;
      letter-spacing: 0.15em;
      text-transform: uppercase;
      margin-bottom: 1.5rem;
    }

    #now-playing {
      font-size: 0.8rem;
      color: #888;
      min-height: 1.2em;
      margin-bottom: 1.5rem;
      text-align: center;
      max-width: 400px;
    }

    audio {
      width: 320px;
    }

    #status {
      font-size: 0.7rem;
      color: #444;
      margin-top: 1rem;
    }
  </style>
</head>
<body>
  <h1>WAIR-FM</h1>
  <div id="tagline">the frequency between frequencies</div>
  <div id="now-playing">connecting...</div>

  <audio id="player" controls autoplay>
    <source src="/stream/wair-fm" type="audio/ogg">
    <source src="/stream/wair-fm" type="audio/mpeg">
    Your browser does not support audio streaming.
  </audio>

  <div id="status"></div>

  <script>
    const player = document.getElementById('player');
    const nowPlaying = document.getElementById('now-playing');
    const status = document.getElementById('status');

    async function updateNowPlaying() {
      try {
        const r = await fetch('/api/now-playing');
        if (!r.ok) throw new Error(r.status);
        const d = await r.json();
        nowPlaying.textContent = d.track
          ? `${d.show_id}  ·  ${d.track.replace(/\.\w+$/, '')}`
          : 'on air';
        status.textContent = '';
      } catch (e) {
        status.textContent = 'station status unavailable';
      }
    }

    player.addEventListener('error', () => {
      nowPlaying.textContent = 'offline';
    });

    player.addEventListener('playing', () => {
      nowPlaying.textContent = nowPlaying.textContent === 'connecting...'
        ? 'on air' : nowPlaying.textContent;
    });

    updateNowPlaying();
    setInterval(updateNowPlaying, 15000);
  </script>
</body>
</html>
```

### Multiple stations

For multiple stations, add a player block per station and point each at its own mount and API port:

```html
<!-- Station B -->
<audio controls>
  <source src="/stream/station_b" type="audio/ogg">
</audio>
```

For the API, add a second nginx `location` block proxying to the second station's port (e.g. `localhost:8002`), or query each station's API directly from the browser using its port if you are on the same network.

---

## Keeping services running

Create systemd unit files so everything restarts on reboot.

### `/etc/systemd/system/go-music-gen.service`

```ini
[Unit]
Description=go-music-gen ACE-Step music generation server
After=network.target

[Service]
User=ubuntu
WorkingDirectory=/home/ubuntu/repo/go-music-gen
ExecStart=/home/ubuntu/repo/go-music-gen/run.sh
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### `/etc/systemd/system/airadio-wair-fm.service`

```ini
[Unit]
Description=AI Radio FM — wair-fm station
After=network.target icecast2.service go-music-gen.service

[Service]
User=ubuntu
WorkingDirectory=/home/ubuntu/repo/ai-radio-fm
EnvironmentFile=/home/ubuntu/repo/ai-radio-fm/stations/.env.shared
EnvironmentFile=/home/ubuntu/repo/ai-radio-fm/stations/wair-fm/.env
ExecStart=/home/ubuntu/repo/ai-radio-fm/airadio start --station wair-fm
Restart=on-failure
RestartSec=15

[Install]
WantedBy=multi-user.target
```

Enable all services:

```bash
sudo systemctl daemon-reload
sudo systemctl enable icecast2 go-music-gen airadio-wair-fm
sudo systemctl start go-music-gen airadio-wair-fm
```

---

## Deployment checklist

- [ ] Provision VPS (4 GB RAM minimum)
- [ ] Point DNS A record at server IP
- [ ] Open ports 22, 80, 443 in firewall; block 8000, 8001
- [ ] Install nginx, certbot, icecast2, espeak-ng, Go, ONNX Runtime
- [ ] Clone ai-radio-fm, go-kokoro-tts, go-music-gen; run ACE-Step setup.sh
- [ ] Build all binaries
- [ ] Configure icecast.xml with strong passwords, bind to 127.0.0.1
- [ ] Run `certbot --nginx -d radio.yourdomain.com`
- [ ] Write nginx config, enable site, reload nginx
- [ ] Create `stations/.env.shared` and `stations/wair-fm/.env` with production values
- [ ] Write systemd unit files, enable and start all services
- [ ] Drop `index.html` in `/var/www/airadio/`
- [ ] Open `https://radio.yourdomain.com` and verify stream plays
