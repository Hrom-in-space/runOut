let mediaRecorder;
let audioChunks = [];
let audioBlob;
let audioUrl;

// Получение доступа к микрофону и инициализация mediaRecorder
navigator.mediaDevices.getUserMedia({ audio: true })
    .then(stream => {
        // TODO: add audio/webm support
        let options = { mimeType: window.firstSupportedMimeType };
        mediaRecorder = new MediaRecorder(stream, options);

        mediaRecorder.ondataavailable = event => {
            audioChunks.push(event.data);
        };

        // Установка обработчика onstop здесь, после инициализации mediaRecorder
        mediaRecorder.onstop = () => {
            audioBlob = new Blob(audioChunks, { type: window.firstSupportedMimeType });
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
        document.getElementById('startRecord').textContent = 'STOP';
    }
}

function stopRecording() {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
        mediaRecorder.stop();
    }
    document.getElementById('startRecord').textContent = 'REC';
}

function sendAudioBlob(audioBlob) {
    fetch('/needs', {
        method: 'POST',
        headers: {
            'Content-Type': window.firstSupportedMimeType
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

document.addEventListener('DOMContentLoaded', function() {
    function getAllSupportedMimeTypes(...mediaTypes) {
        if (!mediaTypes.length) mediaTypes.push('audio');  // Задаем по умолчанию 'audio', если типы не указаны
        const CONTAINERS = ['webm', 'ogg', 'mp3', 'mp4', 'mpeg', 'aac', '3gpp', '3gpp2', '3gp2', 'quicktime', 'flac', 'x-flac', 'wave', 'wav', 'x-wav', 'x-pn-wav', 'not-supported'];

        return CONTAINERS.flatMap(ext =>
            mediaTypes.map(mediaType => `${mediaType}/${ext}`)
        ).filter(variation => MediaRecorder.isTypeSupported(variation));  // Проверяем поддержку типов через MediaRecorder.isTypeSupported
    }

    const supportedMimeTypes = getAllSupportedMimeTypes('audio');  // Получаем поддерживаемые MIME-типы
    console.log('Поддерживаемые MIME-типы для записи аудио без указания кодеков:', supportedMimeTypes);

    // Устанавливаем глобальную переменную
    window.firstSupportedMimeType = supportedMimeTypes.length > 0 ? supportedMimeTypes[0] : null;
    console.log('Первый поддерживаемый MIME-тип:', window.firstSupportedMimeType);
});
