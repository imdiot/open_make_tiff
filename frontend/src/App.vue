<script setup>
import * as vue from 'vue'
import {reactive, ref} from 'vue'

import * as runtime from "../wailsjs/runtime/runtime.js";
import * as api from "../wailsjs/go/manager/Api.js";
import * as models from "../wailsjs/go/models.ts"

window.addEventListener('dragover', e => {
  e.preventDefault();
  e.dataTransfer.dropEffect = 'copy';
});

runtime.OnFileDrop((x, y, paths) => {
  api.Convert(paths);
}, false);

const DEFAULT_MESSAGE = "DRAG-N-DROP MORE FILES HERE";
const PROCESSING_MESSAGE = "PROCESSING FILES... DRAG-N-DROP IS DISABLE";
const FINISHED_MESSAGE = "FINISHED PROCESSING FILES";

const config = reactive({
  disableAdobeDNGConverter: false,
  enableWindowTop: false,
  enableSubfolder: false,
  enableCompression: false,
  iccProfile: "",
  workers: 0,
});
const setting = reactive({
  workerNums: [],
  profiles: [],
  enableAdobeDNGConverter: false
});

const textarea = ref(DEFAULT_MESSAGE);
const textareaRef = ref(null);
const running = ref(false);

vue.onMounted(async () => {
  const config_ = await api.GetConfig();
  config.disableAdobeDNGConverter = config_.disable_adobe_dng_converter || config.disableAdobeDNGConverter;
  config.enableWindowTop = config_.enable_window_top || config.enableWindowTop;
  config.enableSubfolder = config_.enable_subfolder || config.enableSubfolder;
  config.enableCompression = config_.enable_compression || config.enableCompression;
  config.iccProfile = config_.icc_profile || config.iccProfile;
  config.workers = config_.workers || config.workers;

  const setting_ = await api.GetSetting();
  setting.enableAdobeDNGConverter = setting_.enable_adobe_dng_converter || setting.enableAdobeDNGConverter;
  const workerNums = [];
  for (const worker_num of setting_.worker_nums) {
    workerNums.push({
      "value": worker_num.value,
      "label": worker_num.label
    });
  }
  setting.workerNums = workerNums;
  const profiles_ = [];
  for (const profile of setting_.profiles) {
    profiles_.push({
      "value": profile.value,
      "label": profile.label
    });
  }
  setting.profiles = profiles_;

  runtime.EventsOn("omt:convert:started", () => {
    running.value = true;
    textarea.value = PROCESSING_MESSAGE + "\n";
  });

  runtime.EventsOn("omt:convert:finished", () => {
    running.value = false;
    textarea.value += FINISHED_MESSAGE + "\n";
    textarea.value += DEFAULT_MESSAGE + "\n";
  });

  runtime.EventsOn("omt:convert:file:started", (path) => {
    textarea.value += path + "\n";
  });
})

vue.onUnmounted(() => {
  runtime.EventsOff("omt:convert:started");
  runtime.EventsOff("omt:convert:finished");
  runtime.EventsOff("omt:convert:file:started");
});

vue.watch(textarea, () => {
  vue.nextTick(() => {
    const textareaElement = textareaRef.value?.$el.querySelector("textarea");
    if (textareaElement) {
      textareaElement.scrollTop = textareaElement.scrollHeight;
    }
  });
});

const handleConfigChange = async () => {
  const cfg = new models.manager.Config({
    disable_adobe_dng_converter: config.disableAdobeDNGConverter,
    enable_window_top: config.enableWindowTop,
    enable_subfolder: config.enableSubfolder,
    enable_compression: config.enableCompression,
    icc_profile: config.iccProfile,
    workers: config.workers,
  })

  const config_ = await api.SetConfig(cfg);
  config.disableAdobeDNGConverter = config_.disable_adobe_dng_converter || config.disableAdobeDNGConverter;
  config.enableWindowTop = config_.enable_window_top || config.enableWindowTop;
  config.enableSubfolder = config_.enable_subfolder || config.enableSubfolder;
  config.enableCompression = config_.enable_compression || config.enableCompression;
  config.iccProfile = config_.icc_profile || config.iccProfile;
  config.workers = config_.workers || config.workers;
};
</script>

<template>
  <el-container
      @drop.prevent
      @dragenter.prevent
      @dragleave.prevent
      @dragover.prevent
      style="height: 100%"
  >
    <el-main
        style="padding-top: 10px; padding-bottom: 0"
    >
      <el-input
          ref="textareaRef"
          v-model="textarea"
          @focus="(e) => e.target.blur()"
          readonly
          type="textarea"
          resize="none"
          style="height: 100%"
          input-style="height: 100%"
      />
    </el-main>
    <el-footer class="auto-height-footer">
      <el-row>
        <el-col :span="12">
          <el-checkbox
              label="keep window top"
              size="small"
              :disabled="running"
              v-model="config.enableWindowTop"
              @change="handleConfigChange"
          />
        </el-col>
        <el-col :span="12">
          <el-checkbox
              label="lzw compression"
              size="small"
              :disabled="running"
              v-model="config.enableCompression"
              @change="handleConfigChange"
          />
        </el-col>
      </el-row>
      <el-row>
        <el-col :span="12">
          <el-checkbox
              label='put in "make_tiff" subfolder'
              size="small"
              :disabled="running"
              v-model="config.enableSubfolder"
              @change="handleConfigChange"
          />
        </el-col>
        <el-col :span="5">
          <el-text
              v-if="running"
              size="small"
              style="font-weight: 500;color: var(--el-disabled-text-color)"
          >num workers:
          </el-text>
          <el-text
              v-else
              size="small"
              style="font-weight:500"
          >num workers:
          </el-text>
        </el-col>
        <el-col :span="7">
          <el-select
              size="small"
              :disabled="running"
              v-model="config.workers"
              @change="handleConfigChange"
              @focus="(e) => e.target.blur()"
          >
            <el-option
                v-for="item in setting.workerNums"
                :key="item.value"
                :label="item.label"
                :value="item.value"
            />
          </el-select>
        </el-col>
      </el-row>
      <el-row>
        <el-col :span="12">
          <el-checkbox
              v-if="setting.enableAdobeDNGConverter"
              label="without Adobe DNG Converter"
              size="small"
              :disabled="running"
              v-model="config.disableAdobeDNGConverter"
              @change="handleConfigChange"
          />
          <el-checkbox
              v-else
              disabled
              checked
              label="without Adobe DNG Converter"
              size="small"
          />
        </el-col>
        <el-col :span="5">
          <el-text
              v-if="running"
              size="small"
              style="font-weight: 500;color: var(--el-disabled-text-color)"
          >icc profile:
          </el-text>
          <el-text
              v-else
              size="small"
              style="font-weight:500"
          >icc profile:
          </el-text>
        </el-col>
        <el-col :span="7">
          <el-select
              size="small"
              :disabled="running"
              v-model="config.iccProfile"
              :empty-values="[null, undefined]"
              @change="handleConfigChange"
              @focus="(e) => e.target.blur()"
          >
            <el-option
                v-for="item in setting.profiles"
                :key="item.value"
                :label="item.label"
                :value="item.value"
            />
          </el-select>
        </el-col>
      </el-row>
    </el-footer>
  </el-container>
</template>

<style>
.auto-height-footer {
  height: auto !important;
  padding-bottom: 10px;
}
</style>
