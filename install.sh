#!/bin/bash
# ============================================================
#  term-ai — Kurulum Scripti
#  Kullanım: curl -fsSL <url>/install.sh | bash
#  Veya:     bash install.sh
# ============================================================

set -e

BINARY_NAME="term-ai"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/term-ai"
CONFIG_FILE="$CONFIG_DIR/config"
GO_VERSION="1.22.4"
GO_ARCH="linux/amd64"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Renkler
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}"
echo "  ████████╗███████╗██████╗ ███╗   ███╗      █████╗ ██╗"
echo "     ██╔══╝██╔════╝██╔══██╗████╗ ████║     ██╔══██╗██║"
echo "     ██║   █████╗  ██████╔╝██╔████╔██║     ███████║██║"
echo "     ██║   ██╔══╝  ██╔══██╗██║╚██╔╝██║     ██╔══██║██║"
echo "     ██║   ███████╗██║  ██║██║ ╚═╝ ██║     ██║  ██║██║"
echo "     ╚═╝   ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝     ╚═╝  ╚═╝╚═╝"
echo -e "${NC}"
echo -e "  ${BLUE}Linux Terminal AI Asistanı — Kurulum Scripti${NC}"
echo "  ────────────────────────────────────────────"
echo ""

# ─── 1. Go kontrolü ───────────────────────────────────────────
check_go() {
    if command -v go &>/dev/null; then
        echo -e "${GREEN}✅ Go bulundu:${NC} $(go version)"
        return 0
    elif [ -x "/usr/local/go/bin/go" ]; then
        export PATH=$PATH:/usr/local/go/bin
        echo -e "${GREEN}✅ Go bulundu:${NC} $(go version)"
        return 0
    fi
    return 1
}

install_go() {
    echo -e "${YELLOW}⬇️  Go ${GO_VERSION} indiriliyor...${NC}"
    local TARBALL="/tmp/go.tar.gz"
    local ARCH
    ARCH=$(uname -m)
    local GOARCH="amd64"
    [ "$ARCH" = "aarch64" ] && GOARCH="arm64"

    wget -q --show-progress \
        "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" \
        -O "$TARBALL"

    echo -e "${YELLOW}📦 Go kuruluyor (/usr/local/go)...${NC}"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "$TARBALL"
    rm -f "$TARBALL"

    # PATH'e ekle
    if ! grep -q '/usr/local/go/bin' "$HOME/.bashrc"; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> "$HOME/.bashrc"
    fi
    export PATH=$PATH:/usr/local/go/bin
    echo -e "${GREEN}✅ Go ${GO_VERSION} kuruldu.${NC}"
}

if ! check_go; then
    echo -e "${YELLOW}⚠️  Go bulunamadı. Otomatik kurulacak.${NC}"
    install_go
fi

echo ""

# ─── 2. Derleme ───────────────────────────────────────────────
echo -e "${BLUE}🔨 Derleniyor...${NC}"
cd "$PROJECT_DIR"
go build -ldflags="-s -w" -o "$BINARY_NAME" .
echo -e "${GREEN}✅ Derleme tamamlandı.${NC} ($(du -sh $BINARY_NAME | cut -f1) binary)"
echo ""

# ─── 3. Binary kurulumu ───────────────────────────────────────
echo -e "${BLUE}📦 ${INSTALL_DIR}/${BINARY_NAME} konumuna kopyalanıyor...${NC}"
sudo cp "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
echo -e "${GREEN}✅ Binary kuruldu: ${INSTALL_DIR}/${BINARY_NAME}${NC}"
echo ""

# ─── 4. Config dosyası ────────────────────────────────────────
mkdir -p "$CONFIG_DIR"

if [ ! -f "$CONFIG_FILE" ]; then
    # Ollama çalışıyorsa ilk modeli tespit et
    DETECTED_MODEL=""
    if command -v ollama &>/dev/null || curl -s http://localhost:11434/api/tags &>/dev/null; then
        DETECTED_MODEL=$(curl -s http://localhost:11434/api/tags 2>/dev/null \
            | grep -o '"name":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi

    cat > "$CONFIG_FILE" <<EOF
# term-ai yapılandırma dosyası
# Değiştirmek için bu dosyayı düzenleyin.
#
# Geçerli sağlayıcılar: ollama, openai, vllm, azure

PROVIDER=ollama
EOF

    if [ -n "$DETECTED_MODEL" ]; then
        echo "MODEL=$DETECTED_MODEL" >> "$CONFIG_FILE"
        echo -e "${GREEN}✅ Otomatik algılanan model: ${DETECTED_MODEL}${NC}"
    else
        echo "# MODEL=gemma2:2b" >> "$CONFIG_FILE"
    fi

    echo -e "${GREEN}✅ Config oluşturuldu: ${CONFIG_FILE}${NC}"
else
    echo -e "${YELLOW}ℹ️  Config zaten mevcut: ${CONFIG_FILE}${NC}"
fi

echo ""

# ─── 5. PATH kontrolü ─────────────────────────────────────────
if [[ ":$PATH:" != *":/usr/local/bin:"* ]]; then
    echo -e "${YELLOW}⚠️  /usr/local/bin PATH'te yok. ~/.bashrc'ye ekleniyor...${NC}"
    echo 'export PATH=$PATH:/usr/local/bin' >> "$HOME/.bashrc"
fi

# ─── 6. Tamamlandı ────────────────────────────────────────────
echo "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  ${GREEN}🎉 Kurulum tamamlandı!${NC}"
echo "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "  ${CYAN}Kullanım:${NC}"
echo -e "    ${YELLOW}term-ai \"sorunuz\"${NC}"
echo -e "    ${YELLOW}term-ai --list${NC}                ← Yüklü modelleri gör"
echo -e "    ${YELLOW}term-ai --model llama3 \"soru\"${NC} ← Model seç"
echo ""
echo -e "  ${CYAN}Config dosyası:${NC} ${CONFIG_FILE}"
echo ""
echo -e "  ${BLUE}Yeni terminalde çalışır. Şu anki terminal için:${NC}"
echo -e "    ${YELLOW}source ~/.bashrc${NC}"
echo ""
