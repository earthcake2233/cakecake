import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
  ElTag
} from "element-plus";

/** 项目未全量 app.use(ElementPlus)，按需注册模板里用到的组件 */
const components = [
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElOption,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
  ElTag
];

export function setupElementPlus(app) {
  for (const c of components) {
    app.component(c.name, c);
  }
}
