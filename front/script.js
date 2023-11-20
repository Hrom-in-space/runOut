let mediaRecorder;
let audioChunks = [];
let audioBlob;
let audioUrl;

// Получение доступа к микрофону и инициализация mediaRecorder
navigator.mediaDevices.getUserMedia({ audio: true })
    .then(stream => {
        let options = { mimeType: 'audio/webm' };
        mediaRecorder = new MediaRecorder(stream, options);

        mediaRecorder.ondataavailable = event => {
            audioChunks.push(event.data);
        };

        // Установка обработчика onstop здесь, после инициализации mediaRecorder
        mediaRecorder.onstop = () => {
            audioBlob = new Blob(audioChunks, { type: 'audio/webm' });
            audioUrl = URL.createObjectURL(audioBlob);
            audioChunks = [];
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
        mediaRecorder.start();
        document.getElementById('startRecord').textContent = 'Идет запись';
        // document.getElementById('startRecord').style.color = 'red';
        // document.getElementById('startRecord').style.animation = 'blink 1s linear infinite'

    }
}

function stopRecording() {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
        mediaRecorder.stop();
    }
    document.getElementById('startRecord').textContent = 'Записать';
    // document.getElementById('startRecord').style.color = 'white';
    // document.getElementById('startRecord').style.animation = 'none'
}

function sendAudioBlob(audioBlob) {
    fetch('http://localhost:8080/needs', {
        method: 'POST',
        headers: {
            'Content-Type': 'audio/webm'
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
