<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  Table,
  Pagination,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  message,
} from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import { truncate } from '@tcm/shared';
import type { Medicine, MedicineRequest } from '@tcm/api';

const apis = useApi();

const loading = ref(false);
const medicines = ref<Medicine[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10 });
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);

const formRef = ref<FormInstance>();
const formState = reactive<MedicineRequest>({
  name: '',
  pinyin: '',
  alias: [],
  nature: '',
  flavor: '',
  meridian: '',
  efficacy: '',
  dosage: '',
  toxicity: '',
});

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '药名', dataIndex: 'name', key: 'name' },
  { title: '拼音', dataIndex: 'pinyin', key: 'pinyin', width: 140 },
  { title: '性味', dataIndex: 'nature_flavor', key: 'nature_flavor', width: 140 },
  { title: '归经', dataIndex: 'meridian', key: 'meridian', width: 120 },
  { title: '功效', dataIndex: 'efficacy', key: 'efficacy' },
  { title: '毒性', dataIndex: 'toxicity', key: 'toxicity', width: 100 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200, fixed: 'right' as const },
];

const dataSource = computed(() =>
  medicines.value.map((m) => ({
    ...m,
    pinyin: m.pinyin || '—',
    nature_flavor: `${m.nature || ''}${m.flavor || ''}` || '—',
    meridian: m.meridian || '—',
    efficacy: truncate(m.efficacy, 40),
    toxicity: m.toxicity || '—',
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.history.listMedicines({ page: query.page, page_size: query.page_size });
    medicines.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function onPageChange(p: number, ps: number) {
  query.page = p;
  query.page_size = ps;
  load();
}

function parseAlias(): string[] {
  if (Array.isArray(formState.alias)) return formState.alias;
  if (typeof formState.alias === 'string' && formState.alias) {
    return (formState.alias as string)
      .split(/[,，、]/)
      .map((s) => s.trim())
      .filter(Boolean);
  }
  return [];
}

function resetForm() {
  formState.name = '';
  formState.pinyin = '';
  formState.alias = [];
  formState.nature = '';
  formState.flavor = '';
  formState.meridian = '';
  formState.efficacy = '';
  formState.dosage = '';
  formState.toxicity = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Medicine) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.history.getMedicine(record.id);
    formState.name = detail.name;
    formState.pinyin = detail.pinyin;
    formState.alias = Array.isArray(detail.alias_json) ? detail.alias_json : [];
    formState.nature = detail.nature;
    formState.flavor = detail.flavor;
    formState.meridian = detail.meridian;
    formState.efficacy = detail.efficacy;
    formState.dosage = detail.dosage;
    formState.toxicity = detail.toxicity;
  } catch (e) {
    formState.name = record.name;
    formState.pinyin = record.pinyin;
    formState.alias = Array.isArray(record.alias_json) ? record.alias_json : [];
    formState.nature = record.nature;
    formState.flavor = record.flavor;
    formState.meridian = record.meridian;
    formState.efficacy = record.efficacy;
    formState.dosage = record.dosage;
    formState.toxicity = record.toxicity;
  }
  modalVisible.value = true;
}

async function handleOk() {
  try {
    await formRef.value?.validate();
  } catch (e) {
    return;
  }

  modalLoading.value = true;
  try {
    const payload: MedicineRequest = {
      ...formState,
      alias: parseAlias(),
    };
    if (currentId.value) {
      await apis.history.updateMedicine(currentId.value, payload);
      message.success('更新成功');
    } else {
      await apis.history.createMedicine(payload);
      message.success('创建成功');
    }
    modalVisible.value = false;
    load();
  } finally {
    modalLoading.value = false;
  }
}

async function handleDelete(id: number) {
  try {
    await apis.history.deleteMedicine(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">药物管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Button type="primary" @click="openAddModal">新增药物</Button>
      </div>
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
        :scroll="{ x: 1200 }"
      >
        <template #bodyCell="{ text, column, record }">
          <template v-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openEditModal(record as Medicine)"
              >编辑</Button
            >
            <Popconfirm title="确定删除该药物吗？" @confirm="handleDelete((record as Medicine).id)">
              <Button type="link" size="small" danger>删除</Button>
            </Popconfirm>
          </template>
          <template v-else>{{ text }}</template>
        </template>
      </Table>
      <div v-if="total > 0" class="pagination-wrap">
        <Pagination
          :current="query.page"
          :page-size="query.page_size"
          :total="total"
          show-size-changer
          :page-size-options="['10', '20', '50']"
          @change="onPageChange"
        />
      </div>
    </div>

    <Modal
      :open="modalVisible"
      :title="currentId ? '编辑药物' : '新增药物'"
      :confirm-loading="modalLoading"
      :width="640"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item label="药名" name="name" :rules="[{ required: true, message: '请输入药名' }]">
          <Input v-model:value="formState.name" placeholder="请输入药名" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="拼音" name="pinyin">
            <Input v-model:value="formState.pinyin" placeholder="请输入拼音" />
          </Form.Item>
          <Form.Item label="别名" name="alias_text">
            <Input
              :value="Array.isArray(formState.alias) ? formState.alias.join('、') : ''"
              placeholder="多个别名用逗号分隔"
              @input="
                (e: Event) => {
                  formState.alias = (e.target as HTMLInputElement).value
                    .split(/[,，、]/)
                    .map((s) => s.trim())
                    .filter(Boolean);
                }
              "
            />
          </Form.Item>
        </div>
        <div class="form-row">
          <Form.Item label="性味" name="nature">
            <Select v-model:value="formState.nature" placeholder="药性">
              <Select.Option value="寒">寒</Select.Option>
              <Select.Option value="凉">凉</Select.Option>
              <Select.Option value="平">平</Select.Option>
              <Select.Option value="温">温</Select.Option>
              <Select.Option value="热">热</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label="味" name="flavor">
            <Select v-model:value="formState.flavor" placeholder="药味">
              <Select.Option value="酸">酸</Select.Option>
              <Select.Option value="苦">苦</Select.Option>
              <Select.Option value="甘">甘</Select.Option>
              <Select.Option value="辛">辛</Select.Option>
              <Select.Option value="咸">咸</Select.Option>
              <Select.Option value="淡">淡</Select.Option>
              <Select.Option value="涩">涩</Select.Option>
            </Select>
          </Form.Item>
        </div>
        <Form.Item label="归经" name="meridian">
          <Input v-model:value="formState.meridian" placeholder="如：肝经、心经、脾经" />
        </Form.Item>
        <Form.Item label="功效" name="efficacy">
          <Input.TextArea v-model:value="formState.efficacy" placeholder="请输入功效" :rows="3" />
        </Form.Item>
        <div class="form-row">
          <Form.Item label="用量" name="dosage">
            <Input v-model:value="formState.dosage" placeholder="如：3-9g" />
          </Form.Item>
          <Form.Item label="毒性" name="toxicity">
            <Select v-model:value="formState.toxicity" placeholder="毒性">
              <Select.Option value="无毒">无毒</Select.Option>
              <Select.Option value="小毒">小毒</Select.Option>
              <Select.Option value="有毒">有毒</Select.Option>
              <Select.Option value="大毒">大毒</Select.Option>
            </Select>
          </Form.Item>
        </div>
      </Form>
    </Modal>
  </div>
</template>

<style scoped lang="less">
.table-card {
  background-color: #fff;
  border-radius: var(--tcm-radius-lg);
  padding: var(--tcm-spacing-lg);
  box-shadow: var(--tcm-shadow-card);
}

.toolbar {
  margin-bottom: var(--tcm-spacing-base);
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--tcm-spacing-lg);
}

.form-row {
  display: flex;
  gap: var(--tcm-spacing-base);

  .ant-form-item {
    flex: 1;
  }
}
</style>
