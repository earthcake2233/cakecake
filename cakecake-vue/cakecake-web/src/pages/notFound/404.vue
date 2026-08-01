<template>
  <div class="error-container">
    <div class="error-panel server-error error-404">
      <img class="error-sign" :src="verySorry" alt="非常抱歉，你要找的页面不见了" />
      <div class="error-panel__actions">
        <a class="rollback-btn" href="javascript:void(0)" @click.prevent="goBack"
          >返回上一页</a
        >
      </div>
    </div>
    <div class="error-split" aria-hidden="true"></div>
    <div class="error-manga" v-if="mangaList.length">
      <img class="error-manga__img" :src="mangaList[mangaIndex]" alt="" />
      <a
        class="change-img-btn"
        href="javascript:void(0)"
        @click.prevent="changeManga"
        >换一张</a
      >
    </div>
  </div>
</template>

<script>
import verySorry from "@/assets/error/very_sorry.png";
import { ERROR_MANGA_IMAGES } from "@/constants/errorMangaImages";

export default {
  name: "NotFoundPage",
  data() {
    return {
      verySorry,
      mangaList: ERROR_MANGA_IMAGES,
      mangaIndex: 0
    };
  },
  created() {
    this.mangaIndex = Math.floor(Math.random() * this.mangaList.length);
  },
  methods: {
    goBack() {
      if (window.history.length > 1) {
        window.history.back();
        return;
      }
      this.$router.push({ name: "home" });
    },
    changeManga() {
      if (this.mangaList.length <= 1) {
        return;
      }
      let next = this.mangaIndex;
      while (next === this.mangaIndex) {
        next = Math.floor(Math.random() * this.mangaList.length);
      }
      this.mangaIndex = next;
    }
  }
};
</script>

<style lang="scss" scoped>
@import "../../style/mixin";

.error-container {
  width: 1160px;
  margin: 30px auto;
  background: $white;
  @include borderRadius(10px);
}

.error-panel {
  overflow: hidden;

  &.server-error {
    .error-sign {
      display: block;
      margin: 0 auto;
      max-width: 100%;
      height: auto;
    }

    .error-panel__actions {
      padding: 0 0 25px;
      text-align: center;
    }
  }

  .rollback-btn {
    display: inline-block;
    @include wh(140px, 40px);
    margin: 25px auto 0;
    line-height: 40px;
    text-align: center;
    background: $blue;
    @include sc(16px, $white);
    @include borderRadius(4px);
    @include transition(0.2s);
    cursor: pointer;
    text-decoration: none;

    &:hover {
      background: #00b5e5;
    }
  }
}

.error-split {
  @include wh(100%, 40px);
  background: url("../../assets/error/have_rest.png") center no-repeat;
}

.error-manga {
  padding: 30px;
  text-align: center;

  &__img {
    display: block;
    width: 800px;
    max-width: 100%;
    height: auto;
    margin: 0 auto;
  }

  .change-img-btn {
    display: block;
    @include wh(150px, 48px);
    margin: 30px auto 0;
    line-height: 48px;
    text-align: center;
    @include sc(16px, $white);
    background: $blue;
    @include borderRadius(4px);
    @include transition(0.2s);
    cursor: pointer;
    text-decoration: none;

    &:hover {
      background: #00b5e5;
    }
  }
}
</style>
