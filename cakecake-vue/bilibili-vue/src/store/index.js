import { createStore } from "vuex";
import state from "./state";
import * as getters from "./getters";
import * as actions from "./actions";
import mutations from "./mutations";
import modules from "./modules";

const debug = import.meta.env.DEV;

export default createStore({
  state,
  getters,
  mutations,
  actions,
  modules,
  strict: debug
});
