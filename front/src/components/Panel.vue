<script setup>
import {ref, computed, onMounted} from 'vue'
import { needsStore } from '@/stores/counter'

const store = needsStore()

const isRecording = ref(false);
let firstSupportedMimeType;
let microphone;
let recorder;

const recStatus = computed(() => isRecording.value ? "REC" : "STOP");
const classObject = computed(() => ({
  recording: isRecording.value,
  startRecord: true,
}));

function initStream() {
  navigator.mediaDevices.getUserMedia({ audio: {echoCancellation:true} })
      .then(stream => {
        console.log('microphone access: success');
        stream.stop()
      })
      .catch(error => {
        console.error('Error accessing the microphone', error);
      });
}
function sendAudioBlob(audioBlob) {
  fetch('/api/needs', {
    method: 'POST',
    headers: {
      'Content-Type': firstSupportedMimeType
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
function captureMicrophone(callback) {
  if(microphone) {
    callback(microphone);
    return;
  }

  if(typeof navigator.mediaDevices === 'undefined' || !navigator.mediaDevices.getUserMedia) {
    alert('This browser does not supports WebRTC getUserMedia API.');

    if(!!navigator.getUserMedia) {
      alert('This browser seems supporting deprecated getUserMedia API.');
    }
  }

  navigator.mediaDevices.getUserMedia({
    audio: isEdge ? true : {
      echoCancellation: false
    }
  }).then(function(mic) {
    callback(mic);
  }).catch(function(error) {
    alert('Unable to capture your microphone. Please check console logs.');
    console.error(error);
  });
}

function startRecording() {
  captureMicrophone(function(mic) {
    microphone = mic;

    let options = {
      type: 'audio',
      numberOfAudioChannels: isEdge ? 1 : 2,
      checkForInactiveTracks: true,
      bufferSize: 16384
    };

    if (isSafari || isEdge) {
      options.recorderType = StereoAudioRecorder;
    }

    if (navigator.platform && navigator.platform.toString().toLowerCase().indexOf('win') === -1) {
      options.sampleRate = 48000; // or 44100 or remove this line for default
    }

    if (isSafari) {
      options.sampleRate = 44100;
      options.bufferSize = 4096;
      options.numberOfAudioChannels = 2;
    }

    if (recorder) {
      recorder.destroy();
      recorder = null;
    }

    recorder = RecordRTC(microphone, options);

    recorder.startRecording();

    isRecording.value = true;
    isActive.value = true
  });
}

function stopRecording() {
  recorder.stopRecording(function() {
    let blob = recorder.getBlob();
    sendAudioBlob(blob);
    new Audio(URL.createObjectURL(blob)).play();

    console.log('STOP STATE: ', recorder.state);
    isRecording.value = false;

    recorder.destroy();
    recorder = null;

    microphone.stop();
    microphone = null;

    isActive.value = false;

    setTimeout(() => {
      store.allNeeds()
    }, 1000)
    setTimeout(() => {
      store.allNeeds()
    }, 2000)
    setTimeout(() => {
      store.allNeeds()
    }, 3000)
    setTimeout(() => {
      store.allNeeds()
    }, 4000)
    setTimeout(() => {
      store.allNeeds()
    }, 5000)
  });
}

function detectMimeType() {
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
  firstSupportedMimeType = supportedMimeTypes.length > 0 ? supportedMimeTypes[0] : null;
  console.log('Первый поддерживаемый MIME-тип:', firstSupportedMimeType);
}

onMounted(() => {
  detectMimeType()
  initStream()
})

const isActive = ref(false);

</script>

<template>
  <menu
    class="main-menu"
    style="background: url(/image/menu.svg) center/cover no-repeat"
  >
    <div class="menu-buttons">
      <button @click="store.allNeeds" class="menu-btn">
        <img src="/icons/openMenu.svg" class="menu-btn-icon" alt="Меню" />
      </button>
      <button @click="store.clearNeeds" class="menu-btn">
        <img src="/icons/trash.svg" class="menu-btn-icon" alt="Удалить" />
      </button>
    </div>
    <div @touchstart="startRecording" @touchend="stopRecording" class="menu-micro-container" :class="{ active: isActive }">
      <img
        class="menu-micro-active"
        src="/icons/menu-micro-active.svg"
        alt=""
      />
      <div class="menu-micro">
        <img class="menu-micro-icon" src="/icons/micro.svg" alt="" />
      </div>
    </div>
  </menu>
</template>
