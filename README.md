# 🤖 term-ai

**Linux terminali için çok sağlayıcılı, yerel AI asistanı.**

Terminali bilmeden Linux kullanın. Soruyu yazın, AI komutu versin.  
Ollama ile tamamen çevrimdışı ve gizli çalışır.

```
term-ai "nginx logları nerede?"
```
```
🤖 [ollama / gemma4:e4b]

/var/log/nginx/access.log  → erişim logları
/var/log/nginx/error.log   → hata logları

tail -f /var/log/nginx/error.log
```

---

## 📋 İçindekiler

- [Özellikler](#-özellikler)
- [Gereksinimler](#-gereksinimler)
- [Kurulum](#-kurulum)
- [Kullanım Kılavuzu](#-kullanım-kılavuzu)
- [Sağlayıcı Ayarları](#-sağlayıcı-ayarları)
- [Yapılandırma](#-yapılandırma)
- [Gerçek Hayat Örnekleri](#-gerçek-hayat-örnekleri)
- [Mimari](#-mimari)
- [Katkı Sağlama](#-katkı-sağlama)

---

## ✨ Özellikler

| Özellik | Açıklama |
|---|---|
| **Çok Sağlayıcı** | Ollama, OpenAI, vLLM, Azure — tek araçta |
| **Çevrimdışı** | Ollama ile internet gerektirmez |
| **Gizlilik** | Veriler cihazınızdan çıkmaz |
| **Streaming** | Yanıtlar kelime kelime akar |
| **Exec Modu** | AI'nın önerdiği komutu onay alarak çalıştırır |
| **Otomatik Model** | Ollama modelini otomatik algılar |
| **Config Dosyası** | `~/.config/term-ai/config` ile kalıcı ayarlar |
| **Tek Binary** | Kurulumdan sonra tek dosya, sıfır bağımlılık |

---

## ⚙️ Gereksinimler

- Linux (x86_64 veya arm64)
- Go 1.22+ *(install.sh otomatik kurar)*
- En az bir AI sağlayıcı:
  - **Ollama** (önerilen, ücretsiz): https://ollama.com
  - **OpenAI** API anahtarı
  - **vLLM** yerel sunucusu
  - **Azure AI Foundry** endpoint'i

---

## 🚀 Kurulum

### Yöntem 1 — Snap ile (Önerilen, En Kolayı)

Eğer sisteminizde Snap kuruluysa tek komutla yükleyebilirsiniz:

```bash
sudo snap install term-ai --classic
```
> *(Not: Paket Snap Store'a yüklendikten sonra bu komut çalışacaktır.)*

---

### Yöntem 2 — Otomatik Script ile

```bash
git clone https://github.com/seyid12/Term_Ai
cd Term_Ai
bash install.sh
```

`install.sh` şunları otomatik yapar:
- Go yüklü değilse indirir ve kurar
- Kodu derler (5 MB tek binary)
- `/usr/local/bin/term-ai` konumuna kopyalar
- `~/.config/term-ai/config` yapılandırma dosyasını oluşturur
- Ollama varsa ilk modeli otomatik ayarlar

---

### Yöntem 3 — Manuel Kaynaktan Derleme

**Go ile derlemek için:**
```bash
# 1. Go ile derle
go build -ldflags="-s -w" -o term-ai .

# 2. Global yap
sudo install -m 755 term-ai /usr/local/bin/term-ai
```

**Kendi Snap paketinizi derlemek için:**
```bash
snapcraft pack --destructive-mode
sudo snap install term-ai_1.0.0_amd64.snap --dangerous --classic
```

---

### Ollama Kurulumu (Yerel AI için)

```bash
# Ollama'yı kur
curl -fsSL https://ollama.com/install.sh | sh

# Bir model indir (örnekler)
ollama pull gemma2:2b        # Hafif, hızlı (~1.6 GB)
ollama pull gemma4:e2b       # Orta (~7 GB)
ollama pull llama3           # Güçlü (~4.7 GB)
```

---

## 📖 Kullanım Kılavuzu

### Temel Kullanım

```bash
term-ai "sorunuz buraya"
```

Varsayılan olarak Ollama ve otomatik algılanan modeli kullanır.

---

### Tüm Seçenekler

```
term-ai [SEÇENEKLER] "soru"

SEÇENEKLER:
  --provider string   AI sağlayıcı (varsayılan: "ollama")
                      Geçerli değerler: ollama | openai | vllm | azure

  --model string      Model adı (boş bırakılırsa otomatik algılanır)
                      Örnekler: gemma4:e2b, gpt-4o, llama3

  --exec              AI'nın önerdiği bash komutunu onay alarak çalıştırır

  --list              Ollama'daki yüklü modelleri listeler
```

---

### Komut Örnekleri

#### 🔹 Basit soru-cevap

```bash
term-ai "systemd servislerini nasıl listelerim?"
term-ai "en çok RAM kullanan 5 process'i göster"
term-ai "bu klasördeki .log dosyalarını sil"
```

#### 🔹 Yüklü modelleri gör

```bash
term-ai --list
```

Çıktı:
```
🔍 Ollama'daki yüklü modeller:
  1) gemma4:e4b
  2) gemma4:e2b
```

#### 🔹 Farklı model seç

```bash
term-ai --model gemma4:e2b "daha hızlı cevap ver"
term-ai --model llama3 "bash script yaz"
```

#### 🔹 Exec modu — komutu otomatik çalıştır

```bash
term-ai --exec "diskimde kaç GB boş alan var?"
```

Çıktı:
```
🤖 [ollama / gemma4:e4b]

df -h

⚡ Çalıştırılacak komut:
  df -h
   Onaylıyor musunuz? [E/h]: E

📤 Çıktı:
Filesystem      Size  Used Avail Use% Mounted on
/dev/nvme1n1p3  460G  194G  243G  45% /
...
```

> ⚠️ **Güvenlik:** `--exec` sadece güvendiğiniz sorgular için kullanın.  
> Çalıştırmadan önce her komut size gösterilir ve onay istenir.

#### 🔹 Çoklu model karşılaştırma

```bash
# Hızlı cevap için küçük model
term-ai --model gemma4:e2b "iptables kuralını sıfırla"

# Detaylı cevap için büyük model
term-ai --model gemma4:e4b "nginx reverse proxy tam konfigürasyonu"
```

---

## 🔌 Sağlayıcı Ayarları

### Ollama (Varsayılan, Çevrimdışı)

```bash
# Kurulum gerekmiyorsa doğrudan kullan
term-ai "sorunuz"

# Model belirt
term-ai --model llama3 "sorunuz"
```

Ollama'nın çalıştığından emin olun:
```bash
ollama serve          # manuel başlat
systemctl status ollama  # servis durumu
```

---

### OpenAI

```bash
export OPENAI_API_KEY="sk-..."

term-ai --provider openai "sorunuz"
term-ai --provider openai --model gpt-4o "sorunuz"
```

Varsayılan model: `gpt-4o-mini`

---

### vLLM (Kendi Sunucunuz)

```bash
# vLLM sunucunuz http://localhost:8000 portunda çalışıyorsa
term-ai --provider vllm "sorunuz"
term-ai --provider vllm --model "mistralai/Mistral-7B-Instruct" "sorunuz"
```

Varsayılan model: `mistralai/Mistral-7B-Instruct`

---

### Azure AI Foundry

```bash
export AZURE_AI_KEY="..."
export AZURE_AI_ENDPOINT="https://your-resource.services.ai.azure.com/v1"

term-ai --provider azure --model "deployment-adi" "sorunuz"
```

> Azure için `--model` ile **deployment adını** belirtmek zorunludur.

---

## ⚙️ Yapılandırma

### Config Dosyası

`~/.config/term-ai/config` dosyasını düzenleyerek varsayılanları kalıcı olarak değiştirin:

```ini
# term-ai yapılandırma dosyası

# Varsayılan sağlayıcı
PROVIDER=ollama

# Varsayılan model (boş bırakılırsa otomatik algılanır)
MODEL=gemma4:e2b
```

Bu sayede her seferinde `--provider` ve `--model` yazmak zorunda kalmazsınız.

### Ortam Değişkenleri

| Değişken | Açıklama |
|---|---|
| `OPENAI_API_KEY` | OpenAI API anahtarı |
| `AZURE_AI_KEY` | Azure API anahtarı |
| `AZURE_AI_ENDPOINT` | Azure endpoint URL'si |

Kalıcı yapmak için `~/.bashrc` veya `~/.zshrc` dosyanıza ekleyin:

```bash
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.bashrc
source ~/.bashrc
```

---

## 💡 Gerçek Hayat Örnekleri

```bash
# Servis yönetimi
term-ai "nginx'i yeniden başlat"
term-ai "hangi portlar açık?"
term-ai "firewall kurallarını listele"

# Dosya işlemleri
term-ai "30 günden eski .log dosyalarını bul ve sil"
term-ai "bu klasörde en büyük 10 dosyayı listele"
term-ai "bir klasörü rsync ile yedekle"

# Ağ
term-ai "benim dış IP adresim ne?"
term-ai "hangi process 8080 portunu kullanıyor?"
term-ai "DNS sorgusunu nasıl yaparım?"

# Sistem bilgisi
term-ai "CPU ve RAM kullanımını izle"
term-ai "kernel sürümünü öğren"
term-ai "sisteme kaç süredir açık?"

# Geliştirici araçları
term-ai "git log'u güzel formatlı göster"
term-ai "docker container'ları listele ve durumlarını göster"
term-ai "python virtual environment oluştur"

# Script yazdır
term-ai "her gece 02:00'de /home dizinini yedekleyen cron job yaz"
term-ai "disk dolduğunda mail atan bash script yaz"
```

---

## 🏗️ Mimari

```
term-ai
│
├── AIProvider (Interface)
│   ├── OllamaProvider      → /api/chat  (Yerel, çevrimdışı)
│   └── OpenAICompatibleProvider → /chat/completions
│       ├── OpenAI
│       ├── vLLM
│       └── Azure AI Foundry
│
├── Config (~/.config/term-ai/config)
│   └── PROVIDER, MODEL varsayılanları
│
└── main()
    ├── flag.Parse()         → CLI argümanları
    ├── loadConfig()         → Config dosyası
    ├── autoDetectModel()    → Ollama model tespiti
    ├── goroutine            → Arka plan HTTP stream
    │     └── chan<- string  → Token kanalı
    └── select loop          → Anlık ekrana bas
```

**Teknik detaylar:**
- Sıfır harici bağımlılık — sadece Go standart kütüphanesi
- Server-Sent Events (SSE) stream parsing
- `goroutine` + `channel` ile eşzamanlı streaming
- `Interface` ile plug-in mimarisi — yeni sağlayıcı eklemek tek struct

---

## 🔧 Yeni Sağlayıcı Ekleme

`AIProvider` interface'ini implement eden herhangi bir struct otomatik çalışır:

```go
type GroqProvider struct {
    APIKey string
}

func (g *GroqProvider) GenerateStream(
    prompt string,
    model string,
    tokenChan chan<- string,
    errChan chan<- error,
) {
    // Groq OpenAI-uyumlu olduğu için mevcut OpenAICompatibleProvider kullanılabilir
    p := &OpenAICompatibleProvider{
        BaseURL: "https://api.groq.com/openai/v1",
        APIKey:  g.APIKey,
    }
    p.GenerateStream(prompt, model, tokenChan, errChan)
}
```

Ardından `main()` içindeki `switch` bloğuna ekleyin.

---

## 📦 Proje Dosyaları

```
term-ai/
├── main.go       → Tüm uygulama kodu (tek dosya)
├── go.mod        → Go modül tanımı
├── install.sh    → Otomatik kurulum scripti
└── README.md     → Bu döküman
```

---

## 🤝 Katkı Sağlama

1. Fork edin
2. Feature branch oluşturun: `git checkout -b feature/yeni-saglayici`
3. Değişikliklerinizi commit edin: `git commit -m 'Groq sağlayıcısı eklendi'`
4. Push edin: `git push origin feature/yeni-saglayici`
5. Pull Request açın

---

## 📄 Lisans

MIT License — özgürce kullanın, değiştirin, dağıtın.

---

<div align="center">

**term-ai** ile terminali keşfedin 🚀

Sorun mu var? [Issue açın](https://github.com/seyid12/Term_Ai/issues)

</div>
