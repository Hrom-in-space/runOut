let mediaRecorder;
let audioChunks = [];
let audioBlob;
let audioUrl;

// Получение доступа к микрофону и инициализация mediaRecorder
navigator.mediaDevices.getUserMedia({ audio: true })
    .then(stream => {
        // TODO: add audio/webm support
        let options = { mimeType: 'audio/mp4' };
        mediaRecorder = new MediaRecorder(stream, options);

        mediaRecorder.ondataavailable = event => {
            audioChunks.push(event.data);
        };

        // Установка обработчика onstop здесь, после инициализации mediaRecorder
        mediaRecorder.onstop = () => {
            audioBlob = new Blob(audioChunks, { type: 'audio/mp4' });
            audioUrl = URL.createObjectURL(audioBlob);
            audioChunks = [];
            playAudio(audioUrl);
            sendAudioBlob(audioBlob);
        };
    })
    .catch(error => {
        console.error('Ошибка доступа к микрофону:', error);
    });


// let startRecordButton = document.getElementById('startRecord');
document.getElementById('startRecord').addEventListener('click', () => {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
        stopRecording();
        document.getElementById('startRecord').classList.remove('recording');
    } else {
        startRecording();
        document.getElementById('startRecord').classList.add('recording');

    }
});

function startRecording() {
    if (mediaRecorder && mediaRecorder.state !== 'recording') {
        audioChunks = [];
        mediaRecorder.start(1000);
        document.getElementById('startRecord').textContent = 'Идет запись';
    }
}

function stopRecording() {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
        mediaRecorder.stop();
    }
    document.getElementById('startRecord').textContent = 'Записать';
}

function sendAudioBlob(audioBlob) {
    fetch('https://skilled-cockatoo-ghastly.ngrok-free.app/needs', {
        method: 'POST',
        headers: {
            'Content-Type': 'audio/mp4'
        },
        body: audioBlob
    })
        .then(response => response.status)
        .then(data => {
            console.log('Успешная отправка:', data);
        })
        .catch(error => {
            console.error('Ошибка отправки:', error);
        });
}

function playAudio(audioUrl) {
    let audio = new Audio(audioUrl);
    audio.play();
}
