# 🤖 term-ai

**Linux terminali için çok sağlayıcılı, yerel AI asistanı ve Web Tabanlı Terminal (GUI).**

Terminali bilmeden Linux kullanın. Gelişmiş Web Arayüzü sayesinde hem yapay zeka ile sohbet edin hem de anında komutları gömülü terminalde çalıştırın.
Ollama ile tamamen çevrimdışı ve gizli çalışır. İstediğiniz zaman OpenAI veya Gemini'ye geçiş yapın!

---

## ✨ Yeni Nesil Özellikler

| Özellik | Açıklama |
|---|---|
| **Modern Web GUI** | Discord/VS Code benzeri şık tasarım, tek tıkla tarayıcınızda açılır. |
| **Gömülü Terminal** | Tarayıcı içinden gerçek Linux terminalinizi yönetin (xterm.js & PTY). |
| **Çıktı Çözümleme** | Terminaldeki bir hatayı seçip tek tıkla yapay zekaya göndererek çözüm isteyin. |
| **Görsel Ayarlar Menüsü** | API anahtarlarınızı ve modellerinizi arayüzden kolayca değiştirin. |
| **Çok Sağlayıcı** | Ollama, OpenAI, Google Gemini, vLLM, Azure — hepsi tek araçta! |
| **Sıfır Bağımlılık** | HTML/CSS kodları `//go:embed` ile içine gömülüdür. Tek bir binary (exe) olarak çalışır. |

---

## 🚀 Kurulum

### Yöntem 1 — Snap ile (Önerilen)
```bash
sudo snap install term-ai --classic
```

### Yöntem 2 — Kaynaktan Derleme
Sisteminizde Go yüklü olmalıdır.
```bash
git clone https://github.com/seyid12/Term_Ai
cd Term_Ai
go build -ldflags="-s -w" -o term-ai .
sudo install -m 755 term-ai /usr/local/bin/term-ai
```

---

## 📖 Kullanım

Terminalinizden sadece şu komutu çalıştırın:
```bash
term-ai
```
Bu komut arka planda hafif bir yerel sunucu (localhost:8080) başlatır ve varsayılan tarayıcınızda (Chrome, Firefox vb.) otomatik olarak şık Web Arayüzünü açar.

Arayüzde:
- **Sol / Orta Panel:** Yapay zeka ile sohbet edebilir, komut tavsiyeleri alabilirsiniz.
- **Sağ Panel (Canlı Terminal):** Normal bir Linux terminali gibi komut yazabilir ve çalıştırabilirsiniz.
- **Ayarlar:** Sol menüdeki "Ayarlar" butonuna basarak Ollama modellerinizi seçebilir veya OpenAI/Gemini API anahtarlarınızı girip anında kullanıma başlayabilirsiniz. Ayarlarınız `~/.config/term-ai/config` dosyasına şifreli bir şekilde kaydedilir.

---

## 🔌 Sağlayıcılar & Desteklenen Modeller

### 🦙 Ollama (Varsayılan, Çevrimdışı)
Ücretsiz, yerel ve gizlilik odaklı.
- Ayarlar menüsünden "Ollama" seçtiğinizde bilgisayarınızda yüklü tüm modeller otomatik olarak açılır listeye (dropdown) gelir.
- Yüklü modeliniz yoksa terminalden indirebilirsiniz: `ollama pull gemma2:2b`

### 🤖 OpenAI
- Ayarlar menüsünden `sk-...` ile başlayan API anahtarınızı girerek `gpt-4o` veya `gpt-4o-mini` kullanabilirsiniz.

### 🌌 Google Gemini
- Ücretsiz Google AI Studio API anahtarınızı girerek `gemini-2.5-flash` veya `gemini-1.5-pro` modellerini kullanabilirsiniz.

### 🏢 Azure AI & vLLM
- Kurumsal kullanım (Azure) veya yerel sunucu ağı (vLLM) kullananlar için tam destek mevcuttur.

---

## 🏗️ Mimari (Nasıl Çalışıyor?)

```
term-ai (Tek Dosya Binary)
│
├── Web Sunucusu (:8080) & //go:embed static/*
│   ├── index.html, style.css, app.js
│
├── WebSockets
│   ├── /ws/terminal  <-->  Linux PTY (Sanal Terminal / Bash)
│   └── /ws/ai        <-->  Yapay Zeka (Ollama/OpenAI/Gemini Stream)
│
└── Ayarlar (Config)
    └── ~/.config/term-ai/config (Provider, Model ve API Anahtarları)
```

**Sıfır CGO, Sıfır Bağımlılık:** Tüm web sunucusu, PTY entegrasyonu ve WebSocket bağlantıları tamamen Go'nun gücüyle, harici kütüphane kurulumuna (Node.js vb.) ihtiyaç duymadan çalışır.

---

## 🤝 Katkı Sağlama

1. Fork edin
2. Feature branch oluşturun: `git checkout -b feature/yeni-ozellik`
3. Push edin ve Pull Request açın!

---

<div align="center">
**term-ai** ile terminali keşfedin 🚀<br>
Sorun mu var? <a href="https://github.com/seyid12/Term_Ai/issues">Issue açın</a>
</div>
