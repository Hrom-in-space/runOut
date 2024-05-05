let recorder;

(function () {
    var old = console.log;
    var logger = document.getElementById('log');
    console.log = function () {
        for (var i = 0; i < arguments.length; i++) {
            if (typeof arguments[i] == 'object') {
                logger.innerHTML += (JSON && JSON.stringify ? JSON.stringify(arguments[i], undefined, 2) : arguments[i]) ;
            } else {
                logger.innerHTML += " " + arguments[i];
            }
        }
        logger.innerHTML += '<br />'
    }
})();

navigator.mediaDevices.getUserMedia({ audio: {echoCancellation:true} })
    .then(stream => {
        recorder = RecordRTC(stream, {
            type: 'audio',
            mimeType: window.firstSupportedMimeType,
            timeSlice: 1000
        });
    })
    .catch(error => {
        console.error('Error accessing the microphone', error);
    });

document.getElementById('startRecord').addEventListener('click', () => {
    console.log('CLICK STATE: ', recorder.state);
    if (recorder && recorder.state === 'recording') {
        recorder.stopRecording(function() {
            let blob = recorder.getBlob();
            sendAudioBlob(blob);
            new Audio(URL.createObjectURL(blob)).play();
            console.log('STOP STATE: ', recorder.state);
            document.getElementById('startRecord').classList.remove('recording');
            document.getElementById('startRecord').textContent = 'REC'
        });
    } else {
        recorder.startRecording();
        console.log('START STATE: ', recorder.state);
        document.getElementById('startRecord').classList.add('recording');
        document.getElementById('startRecord').textContent = 'STOP';
    }
});

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

document.addEventListener('DOMContentLoaded', function() {
    function getAllSupportedMimeTypes(...mediaTypes) {
        if (!mediaTypes.length) mediaTypes.push('audio');  // Задаем по умолчанию 'audio', если типы не указаны
        const CONTAINERS = ['flac', 'm4a', 'mp3', 'mp4', 'mpeg', 'mpga', 'oga', 'ogg', 'wav', 'webm', 'not-supported'];

        return CONTAINERS.flatMap(ext =>
            mediaTypes.map(mediaType => `${mediaType}/${ext}`)
        ).filter(variation => MediaRecorder.isTypeSupported(variation));  // Проверяем поддержку типов через MediaRecorder.isTypeSupported
    }

    const supportedMimeTypes = getAllSupportedMimeTypes('audio');  // Получаем поддерживаемые MIME-типы
    console.log('Поддерживаемые MIME-типы для записи аудио без указания кодеков:', supportedMimeTypes);

    // Устанавливаем глобальную переменную
    window.firstSupportedMimeType = supportedMimeTypes.length > 0 ? supportedMimeTypes[0] : null;
    console.log('Первый поддерживаемый MIME-тип:', window.firstSupportedMimeType);

    document.getElementById('listNeeds').click();
});

function addStringsToList(needs) {
    // Get the list container
    const listContainer = document.getElementById('listContainer');

    // Clear the existing content
    listContainer.innerHTML = '';

    // Create a new unordered list
    const ul = document.createElement('ul');

    // For each string in the array, create a list item and append it to the unordered list
    needs.forEach(need => {
        const li = document.createElement('li');
        li.textContent = need["name"] + "\u00A0\u00A0\u00A0 ";

        // Create a new button
        const button = document.createElement('button');
        button.textContent = 'Del';
        button.className = 'send-button';
        button.dataset.id = need["id"];

        // Append the button to the list item
        li.appendChild(button);

        // Add event listener to the button
        button.addEventListener('click', function() {
            const taskId = this.getAttribute('data-id');
            fetch('/needs/' + taskId, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json', // Тип содержимого, которое мы отправляем
                },
            })
                .then(response => {
                    if (response.status === 200) {
                        this.parentElement.remove(); // Remove the parent list item
                    }
                    console.log('Delete response:', response)
                })
                .catch((error) => {
                    console.error('Error:', error); // Логируем возможную ошибку
                });
        });

        ul.appendChild(li);
    });

    // Append the unordered list to the list container
    listContainer.appendChild(ul);
}

async function allNeeds() {
    try {
        // Fetch the data from the URL
        const response = await fetch("/needs");

        // Check if the response is ok
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        // Parse the response as JSON
        const data = await response.json();

        // Return the data
        return data;
    } catch (error) {
        console.error('There was an error with the fetch operation: ', error);
    }
}

function clearNeeds() {
    fetch("/needs", {
        method: 'DELETE',
    })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.text();
        })
        .then(data => console.log(data))
        .catch(error => console.error('There was an error with the fetch operation: ', error));
}

document.getElementById('listNeeds').addEventListener('click', async function() {
    const list = await allNeeds();
    addStringsToList(list);
});

document.getElementById('clearNeeds').addEventListener('click', async function() {
    clearNeeds();
    addStringsToList([]);
});

document.querySelectorAll('.send-button').forEach(button => {
    button.addEventListener('click', function() {
        const taskId = this.getAttribute('data-id');
        fetch('/needs/' + taskId, {
            method: 'DELETE',
            headers: {
                'Content-Type': 'application/json', // Тип содержимого, которое мы отправляем
            },
        })
            .then(response => response.json()) // Преобразуем ответ сервера в JSON
            .then(data => {
                console.log('Success:', data); // Логируем успех
            })
            .catch((error) => {
                console.error('Error:', error); // Логируем возможную ошибку
            });
    });
});
