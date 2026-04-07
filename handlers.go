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
  <title>Prispevaj</title>
  <link rel="stylesheet" href="/static/style.css">
</head>
<body class="upload-body">
  <p class="screen-label">Ma_{{.ID}}</p>
  <h1 class="upload-title">Prispevaj svoj delček</h1>
  <p class="upload-subtitle">Izberi trenutek in ga prepusti naključju.<br>Popolno bo prav zato, ker ni načrtovano.</p>
  <div class="pick-buttons">
    <button class="pick-btn" id="btn-camera">Posnami</button>
    <button class="pick-btn" id="btn-gallery">Izberi</button>
  </div>
  <input id="input-camera"  type="file" accept="image/*" capture="environment" hidden>
  <input id="input-gallery" type="file" accept="image/*" hidden>
  <div id="preview-wrap" hidden>
    <img id="preview" src="" alt="">
  </div>
  <p id="disclaimer" hidden>S prenosom fotografije dovoljuješ, da tvoj utrinek postane del skupne projekcije. S tem se strinjaš, da tvoje delo postane neločljiv del razstave, se odpoveduješ avtorskim pravicam v sklopu tega projekta in mi pomagaš dokazati, da je lepota v tistem, česar ne moremo nadzorovati.</p>
  <p id="status" hidden></p>
  <button id="submit-btn" hidden>Naloži</button>
  <script>
    const id = {{.IDJson}};
    const inputCamera  = document.getElementById('input-camera');
    const inputGallery = document.getElementById('input-gallery');
    const previewWrap = document.getElementById('preview-wrap');
    const preview     = document.getElementById('preview');
    const disclaimer  = document.getElementById('disclaimer');
    const submitBtn   = document.getElementById('submit-btn');
    const status      = document.getElementById('status');
    let activeInput = null;

    document.getElementById('btn-camera').addEventListener('click',  () => { activeInput = inputCamera;  inputCamera.click(); });
    document.getElementById('btn-gallery').addEventListener('click', () => { activeInput = inputGallery; inputGallery.click(); });

    [inputCamera, inputGallery].forEach(input => {
      input.addEventListener('change', () => {
        const file = input.files[0];
        if (!file) return;
        activeInput = input;
        preview.src = URL.createObjectURL(file);
        previewWrap.hidden = false;
        disclaimer.hidden = false;
        submitBtn.hidden = false;
        status.hidden = true;
        status.className = '';
      });
    });

    submitBtn.addEventListener('click', async () => {
      const file = activeInput && activeInput.files[0];
      if (!file) return;
      submitBtn.textContent = 'Nalagam...';
      submitBtn.disabled = true;
      const fd = new FormData();
      fd.append('image', file);
      try {
        const res = await fetch('/api/upload/' + id, { method: 'POST', body: fd });
        if (res.ok) {
          inputCamera.value = '';
          inputGallery.value = '';
          previewWrap.hidden = true;
          preview.src = '';
          disclaimer.hidden = true;
          submitBtn.hidden = true;
          submitBtn.disabled = false;
          status.className = 'status-success';
          status.textContent = 'Tvoja fotografija bo kmalu postala del živega organizma te razstave.';
          status.hidden = false;
        } else {
          status.className = 'status-error';
          status.textContent = 'Napaka ' + res.status;
          status.hidden = false;
          submitBtn.disabled = false;
        }
      } catch (e) {
        status.className = 'status-error';
        status.textContent = 'Napaka omrežja';
        status.hidden = false;
        submitBtn.disabled = false;
      }
      submitBtn.textContent = 'Naloži';
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
