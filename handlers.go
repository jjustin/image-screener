package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sort"
	"strings"
)

type Handlers struct {
	store *Store
}

var screenTmpl = template.Must(template.New("screen").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Screen {{.ID}}</title>
  <link rel="stylesheet" href="/static/style.css">
</head>
<body class="screen-body">
  <img id="display" src="/api/image/{{.ID}}?t=0" alt="">
  <script>
    const id = {{.IDJson}};
    const img = document.getElementById('display');
    function refresh() {
      img.src = '/api/image/' + id + '?t=' + Date.now();
      setTimeout(refresh, 10000);
    }
    setTimeout(refresh, 10000);
  </script>
</body>
</html>`))

var uploadTmpl = template.Must(template.New("upload").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Upload to Screen {{.ID}}</title>
  <link rel="stylesheet" href="/static/style.css">
</head>
<body class="upload-body">
  <h1>Screen {{.ID}}</h1>
  <div class="pick-buttons">
    <button class="file-label" id="btn-camera">Take Photo</button>
    <button class="file-label" id="btn-gallery">Choose from Gallery</button>
  </div>
  <div id="preview-wrap">
    <img id="preview" src="" alt="" hidden>
  </div>
  <input id="input-camera"  type="file" accept="image/*" capture="environment" hidden>
  <input id="input-gallery" type="file" accept="image/*" hidden>
  <button id="submit-btn" disabled>Send to Screen</button>
  <p id="status"></p>
  <script>
    const id = {{.IDJson}};
    const inputCamera  = document.getElementById('input-camera');
    const inputGallery = document.getElementById('input-gallery');
    const preview   = document.getElementById('preview');
    const submitBtn = document.getElementById('submit-btn');
    const status    = document.getElementById('status');
    let activeInput = null;

    document.getElementById('btn-camera').addEventListener('click',  () => { activeInput = inputCamera;  inputCamera.click(); });
    document.getElementById('btn-gallery').addEventListener('click', () => { activeInput = inputGallery; inputGallery.click(); });

    [inputCamera, inputGallery].forEach(input => {
      input.addEventListener('change', () => {
        const file = input.files[0];
        if (!file) return;
        activeInput = input;
        preview.src = URL.createObjectURL(file);
        preview.hidden = false;
        submitBtn.disabled = false;
        status.className = '';
        status.textContent = '';
      });
    });

    submitBtn.addEventListener('click', async () => {
      const file = activeInput && activeInput.files[0];
      if (!file) return;
      submitBtn.disabled = true;
      submitBtn.textContent = 'Uploading...';
      const fd = new FormData();
      fd.append('image', file);
      try {
        const res = await fetch('/api/upload/' + id, { method: 'POST', body: fd });
        if (res.ok) {
          status.className = 'success';
          status.textContent = 'Upload successful. You image will be eventually displayed on the screen.';
          inputCamera.value = '';
          inputGallery.value = '';
          preview.hidden = true;
          preview.src = '';
          submitBtn.disabled = true;
        } else {
          status.className = 'error';
          status.textContent = 'Error ' + res.status;
          submitBtn.disabled = false;
        }
      } catch (e) {
        status.className = 'error';
        status.textContent = 'Network error';
        submitBtn.disabled = false;
      }
      submitBtn.textContent = 'Send to Screen';
    });
  </script>
</body>
</html>`))

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>Screens</title>
  <link rel="stylesheet" href="/static/style.css">
</head>
<body class="index-body">
  <h1>Screens</h1>
  <ul>{{range .Screens}}
    <li>
      <a href="/screen/{{.}}">Screen {{.}}</a>
      &mdash; <a href="/api/qr/{{.}}">QR code</a>
    </li>{{end}}
  </ul>
</body>
</html>`))

func (h *Handlers) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	screens := h.store.Screens()
	sort.Strings(screens)
	indexTmpl.Execute(w, map[string]any{"Screens": screens})
}

func (h *Handlers) Screen(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/screen/")
	if !h.store.HasScreen(id) {
		http.NotFound(w, r)
		return
	}
	screenTmpl.Execute(w, map[string]any{
		"ID":     id,
		"IDJson": template.JS(fmt.Sprintf("%q", id)),
	})
}

func (h *Handlers) Upload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/upload/")
	if !h.store.HasScreen(id) {
		http.NotFound(w, r)
		return
	}
	uploadTmpl.Execute(w, map[string]any{
		"ID":     id,
		"IDJson": template.JS(fmt.Sprintf("%q", id)),
	})
}

func (h *Handlers) APIImage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/image/")
	if !h.store.HasScreen(id) {
		http.NotFound(w, r)
		return
	}
	entry, ok := h.store.Random(id)
	if !ok {
		http.Error(w, "no image", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", entry.MIMEType)
	w.Header().Set("Cache-Control", "no-store")
	w.Write(entry.Data)
}

func (h *Handlers) APIUpload(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/upload/")
	if !h.store.HasScreen(id) {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing image field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		http.Error(w, "file is not an image", http.StatusBadRequest)
		return
	}

	if err := h.store.Add(id, data); err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) APIQr(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/qr/")
	if !h.store.HasScreen(id) {
		http.NotFound(w, r)
		return
	}
	h.store.mu.RLock()
	qr := h.store.qrs[id]
	h.store.mu.RUnlock()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(qr)
}
