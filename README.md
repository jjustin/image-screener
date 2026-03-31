# image-screener

A simple digital signage server. Point a browser at it and it shows a rotating slideshow of uploaded photos. Anyone nearby can scan a QR code to upload from their phone.

## How it works

Each screen gets its own URL (`/screen/<id>`) and upload page (`/upload/<id>`). The screen page picks a random image every 10 seconds. If nothing has been uploaded yet, it shows a QR code linking to the upload page instead.

Images are compressed to 1080p on upload and stored on disk, so they survive restarts.

## Running

```
go run . -screens=living-room,kitchen -base-url=http://192.168.1.10:8080
```

Or with Docker:

```
docker run -v ./data:/data ghcr.io/jjustin/image-screener \
  -screens=living-room,kitchen \
  -base-url=http://192.168.1.10:8080 \
  -data-dir=/data
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addr` | `:8080` | Listen address |
| `-base-url` | `http://localhost<addr>` | Base URL used in QR codes |
| `-screens` | `screen1,screen2` | Comma-separated screen IDs |
| `-data-dir` | `data` | Where uploaded images are stored |
| `-admin-password` | *(disabled)* | Password for the admin panel |

## Admin panel

Set `-admin-password` to enable `/admin`. It shows all uploaded images with a delete button for each. Uses HTTP Basic Auth — leave the username blank, enter the password.
