<template>
  <div class="mb-search-soon">
    <img class="mb-search-soon__img" :src="emptyImg" alt="" />
    <p class="mb-search-soon__text">该功能即将开放</p>
  </div>
</template>

<script>
import emptyImg from "@/assets/empty.png";

export default {
  data() {
    return { emptyImg };
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.mb-search-soon {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 360px;
  padding: 48px 16px 64px;
  box-sizing: border-box;
}
.mb-search-soon__img {
  width: min(380px, 88vw);
  height: auto;
  display: block;
  user-select: none;
  pointer-events: none;
}
.mb-search-soon__text {
  margin: 16px 0 0;
  @include sc(14px, #9499a0);
  line-height: 20px;
}
</style>
