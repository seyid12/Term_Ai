// Terminal Kurulumu
const term = new Terminal({
    theme: { background: '#000000', foreground: '#e0e0e0', cursor: '#00ffcc' },
    cursorBlink: true,
    fontFamily: 'monospace',
    fontSize: 14
});
const fitAddon = new FitAddon.FitAddon();
term.loadAddon(fitAddon);
term.open(document.getElementById('terminal-container'));
fitAddon.fit();

window.addEventListener('resize', () => {
    fitAddon.fit();
});

// Terminal WebSocket Bağlantısı
const termProtocol = (window.location.protocol === 'https:') ? 'wss:' : 'ws:';
const termSocket = new WebSocket(`${termProtocol}//${window.location.host}/ws/terminal`);

termSocket.onmessage = (event) => {
    term.write(event.data);
};

term.onData(data => {
    if (termSocket.readyState === WebSocket.OPEN) {
        termSocket.send(data);
    }
});

// Yapay Zeka WebSocket Bağlantısı
const aiSocket = new WebSocket(`${termProtocol}//${window.location.host}/ws/ai`);
let currentAIMessageDiv = null;

aiSocket.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'config') {
        document.getElementById('model-info').innerText = `${data.provider} / ${data.model}`;
    } else if (data.type === 'token') {
        if (!currentAIMessageDiv) {
            currentAIMessageDiv = document.createElement('div');
            currentAIMessageDiv.className = 'message ai';
            document.getElementById('messages').appendChild(currentAIMessageDiv);
        }
        
        // Basit markdown render (Sadece kod blokları için)
        currentAIMessageDiv.dataset.raw = (currentAIMessageDiv.dataset.raw || '') + data.content;
        
        let htmlContent = currentAIMessageDiv.dataset.raw
            .replace(/```(?:bash|sh|json)?\n([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
            .replace(/`(.*?)`/g, '<code>$1</code>')
            .replace(/\n/g, '<br>');
            
        currentAIMessageDiv.innerHTML = htmlContent;
        scrollToBottom();
    } else if (data.type === 'done' || data.type === 'error') {
        if (data.type === 'error') {
            const errDiv = document.createElement('div');
            errDiv.className = 'message ai';
            errDiv.style.borderLeftColor = 'red';
            errDiv.innerText = "Hata: " + data.content;
            document.getElementById('messages').appendChild(errDiv);
        }
        currentAIMessageDiv = null;
    }
};

function scrollToBottom() {
    const msgDiv = document.getElementById('messages');
    msgDiv.scrollTop = msgDiv.scrollHeight;
}

function sendToAI(text) {
    if (!text.trim()) return;
    if (aiSocket.readyState !== WebSocket.OPEN) {
        alert("Bağlantı koptu, sayfayı yenileyin.");
        return;
    }
    
    // Kullanıcı mesajı ekle
    const userDiv = document.createElement('div');
    userDiv.className = 'message user';
    userDiv.innerText = text;
    document.getElementById('messages').appendChild(userDiv);
    scrollToBottom();

    // AI'a yolla
    aiSocket.send(JSON.stringify({ prompt: text }));
}

// Arayüz Etkileşimleri
const aiInput = document.getElementById('ai-input');
const sendBtn = document.getElementById('send-ai-btn');

sendBtn.onclick = () => {
    sendToAI(aiInput.value);
    aiInput.value = '';
};

aiInput.onkeypress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendBtn.click();
    }
};

// Terminal Seçimini Gönder
document.getElementById('send-to-ai-btn').onclick = () => {
    if (term.hasSelection()) {
        const selection = term.getSelection();
        aiInput.value = "Lütfen şu terminal hatasını/çıktısını incele:\n```\n" + selection + "\n```\n\n";
        aiInput.focus();
    } else {
        alert("Lütfen önce terminal ekranından farenizle analiz edilmesini istediğiniz bir metni/hatayı seçin (highlight). Ardından bu butona basın.");
    }
};

// Settings Modal Logic
const settingsModal = document.getElementById('settings-modal');
const openSettingsBtn = document.getElementById('open-settings-btn');
const closeSettingsBtn = document.getElementById('close-settings-btn');
const saveSettingsBtn = document.getElementById('save-settings-btn');

async function fetchOllamaModels() {
    try {
        const res = await fetch('/api/ollama/models');
        if (res.ok) {
            const data = await res.json();
            const datalist = document.getElementById('ollama-models');
            datalist.innerHTML = '';
            if (data.models && data.models.length > 0) {
                data.models.forEach(m => {
                    const opt = document.createElement('option');
                    opt.value = m.name;
                    datalist.appendChild(opt);
                });
            }
        }
    } catch(e) {
        console.log("Ollama bağlantısı kurulamadı veya model yok.");
    }
}

openSettingsBtn.onclick = () => {
    fetchOllamaModels();
    settingsModal.style.display = 'flex';
};
closeSettingsBtn.onclick = () => {
    settingsModal.style.display = 'none';
};

const providerSelect = document.getElementById('settings-provider');
providerSelect.addEventListener('change', () => {
    document.querySelectorAll('.api-key-group').forEach(el => el.style.display = 'none');
    if (providerSelect.value === 'openai') document.getElementById('group-openai').style.display = 'flex';
    if (providerSelect.value === 'gemini') document.getElementById('group-gemini').style.display = 'flex';
    if (providerSelect.value === 'azure') document.getElementById('group-azure').style.display = 'flex';
});

saveSettingsBtn.onclick = async () => {
    const provider = document.getElementById('settings-provider').value;
    const model = document.getElementById('settings-model').value;
    const openai_key = document.getElementById('settings-openai-key').value;
    const gemini_key = document.getElementById('settings-gemini-key').value;
    const azure_endpoint = document.getElementById('settings-azure-endpoint').value;
    const azure_key = document.getElementById('settings-azure-key').value;

    try {
        const res = await fetch('/api/settings', {
            method: 'POST',
            body: JSON.stringify({ provider, model, openai_key, gemini_key, azure_endpoint, azure_key }),
            headers: { 'Content-Type': 'application/json' }
        });

        if (res.ok) {
            document.getElementById('model-info').innerText = `${provider} / ${model || 'otomatik'}`;
            settingsModal.style.display = 'none';
            
            const sysDiv = document.createElement('div');
            sysDiv.className = 'message ai';
            sysDiv.style.borderLeftColor = '#00ffcc';
            sysDiv.innerText = "⚙️ Yapay zeka ayarları başarıyla güncellendi! Yeni model aktif.";
            document.getElementById('messages').appendChild(sysDiv);
            scrollToBottom();
        } else {
            alert("Ayarlar kaydedilemedi!");
        }
    } catch (e) {
        alert("Bağlantı hatası: " + e);
    }
};
