(function () {
  var form = document.querySelector(".upload-form");
  var modal = document.getElementById("upload-modal");
  if (!form || !modal) {
    return;
  }

  var category = form.dataset.category || "android";
  var fileInput = document.getElementById("upload-file");
  var openButtons = document.querySelectorAll("#open-upload-modal, [data-open-upload]");
  var closeButtons = modal.querySelectorAll("[data-close-modal]");
  var modalClose = modal.querySelector(".modal-close");
  var dropzone = document.getElementById("modal-dropzone");
  var idleState = document.getElementById("modal-state-idle");
  var uploadingState = document.getElementById("modal-state-uploading");
  var cancelButton = document.getElementById("upload-cancel");
  var statusEl = document.getElementById("upload-status");
  var percentEl = document.getElementById("upload-percent");
  var barEl = document.getElementById("upload-progress-bar");
  var bytesEl = document.getElementById("upload-bytes");
  var speedEl = document.getElementById("upload-speed");
  var uploadNameEl = document.getElementById("modal-upload-name");
  var uploadSizeEl = document.getElementById("modal-upload-size");
  var uploadIconEl = document.getElementById("modal-upload-icon");
  var dropzoneIconEl = document.getElementById("modal-dropzone-icon");

  var uploading = false;
  var activeXHR = null;
  var lastLoaded = 0;
  var lastTime = 0;

  var categoryIcons = {
    android: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false"><path fill="#3DDC84" d="M17.523 15.3414c-.5511 0-.9993-.4486-.9993-.9997s.4483-.9993.9993-.9993c.5511 0 .9993.4483.9993.9993.0001.5511-.4482.9997-.9993.9997m-11.046 0c-.5511 0-.9993-.4486-.9993-.9997s.4482-.9993.9993-.9993c.5511 0 .9993.4483.9993.9993 0 .5511-.4483.9997-.9993.9997m11.4045-6.02l1.9973-3.4592a.4175.4175 0 0 0-.1527-.5717.4185.4185 0 0 0-.5715.1527l-2.0223 3.503C15.5902 8.2439 13.8533 7.8508 12 7.8508s-3.5901.3931-5.1367 1.0949L4.841 5.4416a.4185.4185 0 0 0-.5715-.1527.4175.4175 0 0 0-.1527.5717L6.1141 9.3211C2.6195 11.1832 0 14.0322 0 17.4297v1.5h24v-1.5c0-3.3975-2.6195-6.2465-6.1185-8.1083"/></svg>',
    ios: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false"><path fill="currentColor" d="M17.05 20.28c-.98.95-2.05.8-3.08.35-1.09-.46-2.09-.48-3.24 0-1.44.62-2.2.44-3.06-.35C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.24 2.31-.93 3.57-.84 1.51.12 2.65.72 3.4 1.8-3.12 1.87-2.38 5.98.48 7.13-.57 1.5-1.31 2.99-2.54 4.09zM12.03 7.25c-.15-2.23 1.66-4.07 3.74-4.25.29 2.58-2.34 4.5-3.74 4.25z"/></svg>',
    file: '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false"><path fill="currentColor" d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>'
  };

  var messages = {
    android: {
      empty: "请选择 APK 文件",
      invalid: "只允许上传 .apk 文件"
    },
    ios: {
      empty: "请选择 IPA 文件",
      invalid: "只允许上传 .ipa 文件"
    },
    file: {
      empty: "请选择要上传的文件",
      invalid: "请上传有效的文件"
    }
  };

  function currentMessages() {
    return messages[category] || messages.android;
  }

  function categoryIcon() {
    return categoryIcons[category] || '<svg viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" focusable="false"><path fill="currentColor" d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>';
  }

  function formatBytes(size) {
    if (!size || size < 0) {
      return "0 B";
    }
    var units = ["B", "KB", "MB", "GB"];
    var value = size;
    var unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
      value /= 1024;
      unitIndex += 1;
    }
    if (unitIndex === 0) {
      return Math.round(value) + " " + units[unitIndex];
    }
    return value.toFixed(1) + " " + units[unitIndex];
  }

  function formatSpeed(bytesPerSecond) {
    if (!bytesPerSecond || bytesPerSecond <= 0) {
      return "";
    }
    return formatBytes(bytesPerSecond) + "/s";
  }

  function isAllowedFile(name) {
    if (!name) {
      return false;
    }
    if (category === "android") {
      return /\.apk$/i.test(name);
    }
    if (category === "ios") {
      return /\.ipa$/i.test(name);
    }
    return /\.[^./\\]+$/i.test(name);
  }

  function setModalOpen(open) {
    modal.classList.toggle("hidden", !open);
    modal.setAttribute("aria-hidden", open ? "false" : "true");
    document.body.classList.toggle("modal-open", open);
  }

  function showIdleState() {
    idleState.classList.remove("hidden");
    uploadingState.classList.add("hidden");
    statusEl.classList.add("hidden");
    statusEl.textContent = "";
    barEl.style.width = "0%";
    barEl.classList.remove("is-indeterminate", "is-error", "is-complete");
    percentEl.textContent = "0%";
    bytesEl.textContent = "0 B / 0 B";
    speedEl.textContent = "";
    lastLoaded = 0;
    lastTime = 0;
  }

  function showUploadingState(file) {
    idleState.classList.add("hidden");
    uploadingState.classList.remove("hidden");
    uploadNameEl.textContent = file.name;
    uploadSizeEl.textContent = formatBytes(file.size);
    uploadIconEl.innerHTML = categoryIcon();
    dropzoneIconEl.innerHTML = categoryIcon();
    modalClose.disabled = true;
    cancelButton.disabled = false;
  }

  function setUploading(active) {
    uploading = active;
    fileInput.disabled = active;
    form.querySelectorAll(".modal-meta input").forEach(function (input) {
      input.disabled = active;
    });
    openButtons.forEach(function (button) {
      button.disabled = active;
    });
    if (!active) {
      modalClose.disabled = false;
    }
  }

  function updateProgress(loaded, total) {
    if (total > 0) {
      var percent = Math.min(100, Math.round((loaded / total) * 100));
      barEl.classList.remove("is-indeterminate");
      barEl.style.width = percent + "%";
      percentEl.textContent = percent + "%";
      bytesEl.textContent = formatBytes(loaded) + " / " + formatBytes(total);

      var now = Date.now();
      if (lastTime > 0) {
        var deltaBytes = loaded - lastLoaded;
        var deltaTime = (now - lastTime) / 1000;
        if (deltaTime > 0 && deltaBytes >= 0) {
          speedEl.textContent = formatSpeed(deltaBytes / deltaTime);
        }
      }
      lastLoaded = loaded;
      lastTime = now;
      return;
    }

    barEl.classList.add("is-indeterminate");
    barEl.style.width = "100%";
    percentEl.textContent = "上传中";
    bytesEl.textContent = formatBytes(loaded) + " / 未知大小";
    speedEl.textContent = "";
  }

  function showStatus(message, isError) {
    statusEl.textContent = message;
    statusEl.classList.remove("hidden");
    statusEl.classList.toggle("is-error", !!isError);
  }

  function resetModal() {
    if (activeXHR) {
      activeXHR.abort();
      activeXHR = null;
    }
    setUploading(false);
    showIdleState();
    fileInput.value = "";
  }

  function openModal() {
    if (uploading) {
      return;
    }
    showIdleState();
    setModalOpen(true);
  }

  function closeModal(force) {
    if (uploading && !force) {
      return;
    }
    resetModal();
    setModalOpen(false);
  }

  function startUpload(file) {
    var msg = currentMessages();
    if (!file) {
      window.alert(msg.empty);
      return;
    }
    if (!isAllowedFile(file.name)) {
      window.alert(msg.invalid);
      fileInput.value = "";
      return;
    }

    var formData = new FormData(form);
    var xhr = new XMLHttpRequest();
    activeXHR = xhr;

    setUploading(true);
    showUploadingState(file);
    updateProgress(0, file.size);

    xhr.open("POST", form.action || "/upload", true);
    xhr.upload.addEventListener("progress", function (event) {
      updateProgress(event.loaded, event.lengthComputable ? event.total : file.size);
    });

    xhr.addEventListener("load", function () {
      activeXHR = null;
      if (xhr.status >= 200 && xhr.status < 400) {
        barEl.classList.add("is-complete");
        barEl.style.width = "100%";
        percentEl.textContent = "100%";
        showStatus("上传成功，正在刷新页面…", false);
        window.setTimeout(function () {
          window.location.href = xhr.responseURL || "/?tab=" + category;
        }, 450);
        return;
      }

      barEl.classList.add("is-error");
      showStatus("上传失败，请重试", true);
      setUploading(false);
      cancelButton.disabled = false;
    });

    xhr.addEventListener("error", function () {
      activeXHR = null;
      barEl.classList.add("is-error");
      showStatus("网络错误，上传失败", true);
      setUploading(false);
      cancelButton.disabled = false;
    });

    xhr.addEventListener("abort", function () {
      activeXHR = null;
      resetModal();
    });

    xhr.send(formData);
  }

  openButtons.forEach(function (button) {
    button.addEventListener("click", openModal);
  });

  closeButtons.forEach(function (button) {
    button.addEventListener("click", function () {
      closeModal(false);
    });
  });

  cancelButton.addEventListener("click", function () {
    if (activeXHR) {
      activeXHR.abort();
    } else {
      resetModal();
    }
  });

  fileInput.addEventListener("change", function () {
    var file = fileInput.files && fileInput.files[0];
    if (file) {
      startUpload(file);
    }
  });

  ["dragenter", "dragover"].forEach(function (eventName) {
    dropzone.addEventListener(eventName, function (event) {
      event.preventDefault();
      dropzone.classList.add("is-dragover");
    });
  });

  ["dragleave", "drop"].forEach(function (eventName) {
    dropzone.addEventListener(eventName, function (event) {
      event.preventDefault();
      dropzone.classList.remove("is-dragover");
    });
  });

  dropzone.addEventListener("drop", function (event) {
    if (uploading) {
      return;
    }
    var files = event.dataTransfer && event.dataTransfer.files;
    if (!files || !files.length) {
      return;
    }
    fileInput.files = files;
    startUpload(files[0]);
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && !modal.classList.contains("hidden")) {
      closeModal(false);
    }
  });

  dropzoneIconEl.innerHTML = categoryIcon();
})();
