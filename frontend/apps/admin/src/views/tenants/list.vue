<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  Table,
  Pagination,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  Tag,
  Popconfirm,
  message,
} from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';

import { useApi } from '@/composables/useApi';
import type {
  Tenant,
  CreateTenantRequest,
  UpdateTenantRequest,
  TenantMember,
  AddMemberRequest,
} from '@tcm/api';

const apis = useApi();

// ---- 租户列表 ----
const loading = ref(false);
const tenants = ref<Tenant[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 10, status: '' });

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '机构名称', dataIndex: 'name', key: 'name' },
  { title: '编码', dataIndex: 'code', key: 'code', width: 140 },
  { title: '套餐', dataIndex: 'plan', key: 'plan', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '最大用户数', dataIndex: 'max_users', key: 'max_users', width: 120 },
  { title: '到期时间', dataIndex: 'expires_at', key: 'expires_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 200 },
];

const planOptions = [
  { label: '基础版', value: 'standard' },
  { label: '高级版', value: 'premium' },
  { label: '企业版', value: 'enterprise' },
];

const statusOptions = [
  { label: '全部', value: '' },
  { label: '活跃', value: 'active' },
  { label: '已暂停', value: 'suspended' },
  { label: '已过期', value: 'expired' },
];

const planMap: Record<string, string> = {
  standard: '基础版',
  premium: '高级版',
  enterprise: '企业版',
};
const statusColorMap: Record<string, string> = {
  active: 'green',
  suspended: 'orange',
  expired: 'red',
};
const statusLabelMap: Record<string, string> = {
  active: '活跃',
  suspended: '已暂停',
  expired: '已过期',
};

const dataSource = computed(() =>
  tenants.value.map((t) => ({
    ...t,
    plan_text: planMap[t.plan] || t.plan,
    status_text: statusLabelMap[t.status] || t.status,
    expires_at: t.expires_at || '永久',
  })),
);

async function load() {
  loading.value = true;
  try {
    const res = await apis.tenant.list({
      page: query.page,
      page_size: query.page_size,
      status: query.status,
    });
    tenants.value = res.items ?? [];
    total.value = res.total ?? 0;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  load();
});

function onPageChange(p: number, ps: number) {
  query.page = p;
  query.page_size = ps;
  load();
}

function onStatusChange() {
  query.page = 1;
  load();
}

// ---- 新增/编辑租户 Modal ----
const modalVisible = ref(false);
const modalLoading = ref(false);
const currentId = ref<number | null>(null);
const formRef = ref<FormInstance>();
const formState = reactive<CreateTenantRequest>({
  name: '',
  code: '',
  plan: 'standard',
  max_users: 0,
  expires_at: '',
});

// 编辑模式下的状态字段（仅更新时可修改）
const editStatus = ref<string>('');

function resetForm() {
  formState.name = '';
  formState.code = '';
  formState.plan = 'standard';
  formState.max_users = 0;
  formState.expires_at = '';
  editStatus.value = '';
  formRef.value?.clearValidate();
}

function openAddModal() {
  currentId.value = null;
  resetForm();
  modalVisible.value = true;
}

async function openEditModal(record: Tenant) {
  currentId.value = record.id;
  resetForm();
  try {
    const detail = await apis.tenant.get(record.id);
    formState.name = detail.name;
    formState.code = detail.code;
    formState.plan = detail.plan;
    formState.max_users = detail.max_users;
    formState.expires_at = detail.expires_at || '';
    editStatus.value = detail.status;
  } catch (e) {
    formState.name = record.name;
    formState.code = record.code;
    formState.plan = record.plan;
    formState.max_users = record.max_users;
    formState.expires_at = record.expires_at || '';
    editStatus.value = record.status;
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
    if (currentId.value) {
      const payload: UpdateTenantRequest = {
        name: formState.name,
        plan: formState.plan,
        max_users: formState.max_users,
        expires_at: formState.expires_at || undefined,
      };
      if (editStatus.value) {
        payload.status = editStatus.value;
      }
      await apis.tenant.update(currentId.value, payload);
      message.success('更新成功');
    } else {
      await apis.tenant.create(formState);
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
    await apis.tenant.delete(id);
    message.success('删除成功');
    load();
  } catch (e) {
    // error handled by interceptor
  }
}

// ---- 成员管理 ----
const memberModalVisible = ref(false);
const memberLoading = ref(false);
const currentTenant = ref<Tenant | null>(null);
const members = ref<TenantMember[]>([]);
const memberColumns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '用户 ID', dataIndex: 'user_id', key: 'user_id', width: 120 },
  { title: '角色', dataIndex: 'role', key: 'role', width: 120 },
  { title: '加入时间', dataIndex: 'joined_at', key: 'joined_at', width: 180 },
  { title: '到期时间', dataIndex: 'expired_at', key: 'expired_at', width: 180 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 100 },
];

const memberRoleMap: Record<string, string> = {
  school_admin: '学校管理员',
  teacher: '教师',
  student: '学生',
};

const memberRoleOptions = [
  { label: '学校管理员', value: 'school_admin' },
  { label: '教师', value: 'teacher' },
  { label: '学生', value: 'student' },
];

const memberDataSource = computed(() =>
  members.value.map((m) => ({
    ...m,
    role_text: memberRoleMap[m.role] || m.role,
    expired_at: m.expired_at || '永久',
  })),
);

async function openMemberModal(record: Tenant) {
  currentTenant.value = record;
  memberModalVisible.value = true;
  await loadMembers(record.id);
}

async function loadMembers(tenantId: number) {
  memberLoading.value = true;
  try {
    const res = await apis.tenant.listMembers(tenantId);
    members.value = res.items ?? [];
  } finally {
    memberLoading.value = false;
  }
}

async function handleRemoveMember(userId: number) {
  if (!currentTenant.value) return;
  try {
    await apis.tenant.removeMember(currentTenant.value.id, userId);
    message.success('已移除成员');
    loadMembers(currentTenant.value.id);
  } catch (e) {
    // error handled by interceptor
  }
}

// ---- 添加成员 Modal ----
const addMemberModalVisible = ref(false);
const addMemberLoading = ref(false);
const addMemberFormRef = ref<FormInstance>();
const addMemberForm = reactive<AddMemberRequest>({
  user_id: 0,
  role: 'student',
  expires_at: '',
});

function openAddMemberModal() {
  addMemberForm.user_id = 0;
  addMemberForm.role = 'student';
  addMemberForm.expires_at = '';
  addMemberModalVisible.value = true;
}

async function handleAddMember() {
  try {
    await addMemberFormRef.value?.validate();
  } catch (e) {
    return;
  }
  if (!currentTenant.value) return;

  addMemberLoading.value = true;
  try {
    await apis.tenant.addMember(currentTenant.value.id, {
      user_id: addMemberForm.user_id,
      role: addMemberForm.role,
      expires_at: addMemberForm.expires_at || undefined,
    });
    message.success('添加成员成功');
    addMemberModalVisible.value = false;
    loadMembers(currentTenant.value.id);
  } finally {
    addMemberLoading.value = false;
  }
}
</script>

<template>
  <div class="admin-page">
    <h1 class="admin-page-title">学校/机构管理</h1>

    <div class="table-card">
      <div class="toolbar">
        <Select v-model:value="query.status" style="width: 140px" @change="onStatusChange">
          <Select.Option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </Select.Option>
        </Select>
        <Button type="primary" @click="openAddModal">新增机构</Button>
      </div>
      <Table
        :data-source="dataSource"
        :columns="columns"
        :loading="loading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ text, column, record }">
          <template v-if="column.dataIndex === 'plan'">
            <span>{{ planMap[(record as Tenant).plan] || (record as Tenant).plan }}</span>
          </template>
          <template v-else-if="column.dataIndex === 'status'">
            <Tag :color="statusColorMap[(record as Tenant).status] || 'default'">
              {{ statusLabelMap[(record as Tenant).status] || (record as Tenant).status }}
            </Tag>
          </template>
          <template v-else-if="column.dataIndex === 'max_users'">
            <span>{{
              (record as Tenant).max_users > 0 ? (record as Tenant).max_users : '不限'
            }}</span>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Button type="link" size="small" @click="openEditModal(record as Tenant)">编辑</Button>
            <Button type="link" size="small" @click="openMemberModal(record as Tenant)"
              >成员</Button
            >
            <Popconfirm
              title="确定删除该机构吗？删除后成员关系将一并移除。"
              @confirm="handleDelete((record as Tenant).id)"
            >
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

    <!-- 新增/编辑租户 Modal -->
    <Modal
      :open="modalVisible"
      :title="currentId ? '编辑机构' : '新增机构'"
      :confirm-loading="modalLoading"
      @cancel="modalVisible = false"
      @ok="handleOk"
    >
      <Form ref="formRef" :model="formState" layout="vertical">
        <Form.Item
          label="机构名称"
          name="name"
          :rules="[{ required: true, message: '请输入机构名称' }]"
        >
          <Input v-model:value="formState.name" placeholder="如：北京中医药大学" />
        </Form.Item>
        <Form.Item
          label="机构编码"
          name="code"
          :rules="[{ required: true, message: '请输入机构编码' }]"
        >
          <Input v-model:value="formState.code" placeholder="如：bucm" :disabled="!!currentId" />
        </Form.Item>
        <Form.Item label="套餐" name="plan" :rules="[{ required: true, message: '请选择套餐' }]">
          <Select v-model:value="formState.plan">
            <Select.Option v-for="opt in planOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item v-if="currentId" label="状态" name="status">
          <Select v-model:value="editStatus">
            <Select.Option value="active">活跃</Select.Option>
            <Select.Option value="suspended">已暂停</Select.Option>
            <Select.Option value="expired">已过期</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="最大用户数" name="max_users">
          <InputNumber
            v-model:value="formState.max_users"
            placeholder="0 表示不限"
            style="width: 100%"
            :min="0"
          />
        </Form.Item>
        <Form.Item label="到期时间" name="expires_at">
          <Input
            v-model:value="formState.expires_at"
            placeholder="留空表示永久，如 2027-12-31T23:59:59Z"
          />
        </Form.Item>
      </Form>
    </Modal>

    <!-- 成员管理 Modal -->
    <Modal
      :open="memberModalVisible"
      :title="`成员管理 - ${currentTenant?.name ?? ''}`"
      :footer="null"
      width="720px"
      @cancel="memberModalVisible = false"
    >
      <div class="toolbar">
        <Button type="primary" @click="openAddMemberModal">添加成员</Button>
      </div>
      <Table
        :data-source="memberDataSource"
        :columns="memberColumns"
        :loading="memberLoading"
        :pagination="false"
        row-key="id"
        size="middle"
      >
        <template #bodyCell="{ text, column, record }">
          <template v-if="column.dataIndex === 'role'">
            <Tag>{{
              memberRoleMap[(record as TenantMember).role] || (record as TenantMember).role
            }}</Tag>
          </template>
          <template v-else-if="column.dataIndex === 'action'">
            <Popconfirm
              title="确定移除该成员吗？"
              @confirm="handleRemoveMember((record as TenantMember).user_id)"
            >
              <Button type="link" size="small" danger>移除</Button>
            </Popconfirm>
          </template>
          <template v-else>{{ text }}</template>
        </template>
      </Table>
    </Modal>

    <!-- 添加成员 Modal -->
    <Modal
      :open="addMemberModalVisible"
      title="添加成员"
      :confirm-loading="addMemberLoading"
      @cancel="addMemberModalVisible = false"
      @ok="handleAddMember"
    >
      <Form ref="addMemberFormRef" :model="addMemberForm" layout="vertical">
        <Form.Item
          label="用户 ID"
          name="user_id"
          :rules="[{ required: true, message: '请输入用户 ID' }]"
        >
          <InputNumber
            v-model:value="addMemberForm.user_id"
            placeholder="请输入用户 ID"
            style="width: 100%"
            :min="1"
          />
        </Form.Item>
        <Form.Item label="角色" name="role" :rules="[{ required: true, message: '请选择角色' }]">
          <Select v-model:value="addMemberForm.role">
            <Select.Option v-for="opt in memberRoleOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </Select.Option>
          </Select>
        </Form.Item>
        <Form.Item label="到期时间" name="expires_at">
          <Input v-model:value="addMemberForm.expires_at" placeholder="留空表示永久" />
        </Form.Item>
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
  display: flex;
  gap: var(--tcm-spacing-base);
  margin-bottom: var(--tcm-spacing-base);
}

.pagination-wrap {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--tcm-spacing-lg);
}
</style>
