<template>
  <div class="msg-page">
    <div class="msg-bg" :style="msgBgLayerStyle" aria-hidden="true" />
    <div class="msg-root">
      <div v-if="!token" class="msg-login-hint bili-wrapper">
        <div class="msg-login-card">
          <p class="msg-login-card__crumbs">
            <router-link class="msg-aside__crumb" to="/">主站</router-link>
            <span class="msg-aside__crumb-sep" aria-hidden="true">·</span>
            <router-link class="msg-aside__crumb" to="/minibili/account">
              个人中心
            </router-link>
          </p>
          <p>
            请先
            <a href="#" class="msg-login-card__link" @click.prevent="openLoginModal"
              >登录</a
            >
            后查看消息。
          </p>
        </div>
      </div>

      <div v-else class="msg-layout">
        <div class="msg-stack">
          <div class="msg-panel">
            <aside class="msg-aside">
              <div class="msg-aside__head">
                <span class="msg-aside__plane" aria-hidden="true">
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                    <path
                      d="M22 2L11 13M22 2l-7 20-4-9-9-4 20-7z"
                      stroke="#18191c"
                      stroke-width="1.75"
                      stroke-linejoin="round"
                    />
                  </svg>
                </span>
                <span class="msg-aside__head-text">消息中心</span>
              </div>
              <nav class="msg-aside__nav" aria-label="消息分类">
                <button
                  v-for="item in sidebarNav"
                  :key="item.cat"
                  type="button"
                  class="msg-aside__link"
                  :class="{ 'msg-aside__link--on': activeCat === item.cat }"
                  @click="onSidebarClick(item.cat)"
                >
                  <span class="msg-aside__link-label">{{ item.label }}</span>
                  <span
                    v-if="formatMessageUnreadBadge(msgUnread[item.cat])"
                    class="msg-aside__badge"
                  >{{ formatMessageUnreadBadge(msgUnread[item.cat]) }}</span>
                </button>
                <button
                  type="button"
                  class="msg-aside__link msg-aside__link--settings"
                >
                  <span class="msg-aside__gear" aria-hidden="true">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
                      <path
                        d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"
                        stroke="#61666d"
                        stroke-width="1.5"
                      />
                      <path
                        d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 8 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.6 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 8a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 8 4.6a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 8c.09.33.14.69.14 1.04s-.05.71-.14 1.04a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
                        stroke="#61666d"
                        stroke-width="1.25"
                        stroke-linecap="round"
                        stroke-linejoin="round"
                      />
                    </svg>
                  </span>
                  消息设置
                </button>
              </nav>
            </aside>

            <div class="msg-main">
              <div
                v-show="!isLikeDetailView"
                class="msg-main__title-card"
              >
                <header class="msg-main__bar">
                  <h1 class="msg-main__title">{{ mainTitle }}</h1>
                </header>
              </div>

              <div
                class="msg-main__body-card"
                :class="{
                  'msg-main__body-card--bare':
                    !showMessagesBodyChrome || isLikeDetailView
                }"
              >
                <div
                  v-if="showMsgCenterPageLoading"
                  class="msg-loading-strip"
                  role="status"
                  aria-live="polite"
                >
                  <span class="msg-sr-only">加载中</span>
                  <div class="msg-loading-strip__card">
                    <img
                      class="msg-loading-strip__tv"
                      :src="loadingTvImg"
                      alt=""
                      width="64"
                      height="64"
                    />
                  </div>
                </div>

                <!-- 回复我的：主站单列列表（AC-12 / SPEC NF-9） -->
                <template v-else-if="activeCat === 'reply_received'">
                  <div class="msg-reply-scroll">
                    <ul
                      v-if="replyRows.length"
                      class="msg-reply-list"
                      role="list"
                    >
                      <li
                        v-for="row in replyRows"
                        :key="row.id"
                        class="msg-reply-row"
                        :class="{ 'msg-reply-row--unread': !row.is_read }"
                        @click="onReplyRowShellClick(row, $event)"
                      >
                        <div class="msg-reply-row__cols">
                        <button
                          type="button"
                          class="msg-reply-row__face msg-reply-row__face--tap"
                          :aria-label="`${row.name} 的主页（即将开放）`"
                          @click.stop="onReplySenderProfile(row)"
                        >
                          <img :src="row.face" alt="" />
                        </button>
                        <div class="msg-reply-row__mid">
                          <div class="msg-reply-row__line1">
                            <button
                              type="button"
                              class="msg-reply-row__name msg-reply-row__name--tap"
                              :aria-label="`${row.name} 的主页（即将开放）`"
                              @click.stop="onReplySenderProfile(row)"
                            >
                              {{ row.name }}
                            </button>
                            <span class="msg-reply-row__suffix">{{
                              row.suffix
                            }}</span>
                          </div>
                          <p class="msg-reply-row__content" :title="row.replyText">
                            {{ row.replyTextDisplay }}
                          </p>
                          <div class="msg-reply-row__meta">
                            <time class="msg-reply-row__time">{{
                              row.timeZh
                            }}</time>
                            <button
                              type="button"
                              class="msg-reply-row__act"
                              @click.stop="onToggleReplyComposer(row)"
                            >
                              <svg
                                class="msg-reply-row__ico"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="1.5"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                aria-hidden="true"
                              >
                                <path
                                  d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z"
                                />
                              </svg>
                              回复
                            </button>
                            <button
                              type="button"
                              class="msg-reply-row__act"
                              :class="{
                                'msg-reply-row__act--liked': row.liked_by_me
                              }"
                              @click.stop="onLikeReply(row)"
                            >
                              <svg
                                v-if="!row.liked_by_me"
                                class="msg-reply-row__ico msg-reply-row__ico--like-outline"
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                stroke-width="1.5"
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                aria-hidden="true"
                              >
                                <path
                                  d="M7 10v12M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.67 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2a3.13 3.13 0 0 1 3 3.88Z"
                                />
                              </svg>
                              <svg
                                v-else
                                class="msg-reply-row__ico msg-reply-row__ico--like-solid"
                                viewBox="0 0 24 24"
                                fill="currentColor"
                                aria-hidden="true"
                              >
                                <path
                                  d="M1 21h4V9H1v12zm22-11c0-1.1-.9-2-2-2h-6.31l.95-4.57.03-.32c0-.41-.17-.79-.44-1.06L14.17 1 7.59 7.59C7.22 7.95 7 8.45 7 9v10c0 1.1.9 2 2 2h9c.83 0 1.54-.5 1.84-1.22l3.02-7.05c.09-.23.14-.47.14-.73v-2z"
                                />
                              </svg>
                              {{ row.liked_by_me ? "已赞" : "点赞" }}
                            </button>
                            <button
                              type="button"
                              class="msg-reply-row__act msg-reply-row__act--del"
                              @click.stop="onDeleteNotif(row)"
                            >
                              删除该通知
                            </button>
                          </div>
                        </div>
                        <div
                          class="msg-reply-row__preview"
                          :class="{
                            'msg-reply-row__preview--media': row.isMediaComment
                          }"
                          :title="row.isMediaComment ? '' : row.preview"
                        >
                          <template v-if="row.isMediaComment">
                            <button
                              type="button"
                              class="msg-reply-row__thumb-hit"
                              :aria-label="
                                row.isVideoComment
                                  ? '查看视频内评论'
                                  : '查看专栏内评论'
                              "
                              @click.stop="
                                row.isVideoComment
                                  ? onGoToVideoComment(row)
                                  : onGoToArticleComment(row)
                              "
                            >
                              <span
                                v-if="row.coverUrl"
                                class="msg-reply-row__thumb-wrap"
                              >
                                <img
                                  class="msg-reply-row__thumb"
                                  :src="row.coverUrl"
                                  alt=""
                                />
                              </span>
                              <span
                                v-else
                                class="msg-reply-row__thumb-wrap msg-reply-row__thumb-wrap--empty"
                                aria-hidden="true"
                              />
                            </button>
                          </template>
                          <template v-else>{{ row.previewDisplay }}</template>
                        </div>
                        </div>
                        <div
                          v-if="replyComposerNotifId === row.id"
                          class="msg-reply-compose"
                          @click.stop
                        >
                          <div class="msg-reply-compose__lead">
                            <div class="msg-reply-compose__face">
                              <img :src="meReplyFace" alt="" />
                            </div>
                          </div>
                          <div class="msg-reply-compose__field-wrap">
                            <textarea
                              v-model="replyComposerText"
                              class="msg-reply-compose__textarea"
                              rows="3"
                              maxlength="1000"
                              :placeholder="replyPlaceholder"
                            />
                          </div>
                          <button
                            type="button"
                            class="msg-reply-compose__submit"
                            :disabled="replySubmitting"
                            @click.stop="submitInlineReply(row)"
                          >
                            发表评论
                          </button>
                        </div>
                      </li>
                    </ul>
                    <div v-else class="msg-no-data" aria-live="polite">
                      <div class="msg-no-data__panel">
                        <img
                          class="msg-no-data__img"
                          :src="emptyNoDataImg"
                          alt="然而并没有数据"
                        />
                      </div>
                    </div>
                    <div v-if="nextCursor && !listLoading" class="msg-reply-more">
                      <button
                        type="button"
                        class="msg-reply-more__btn"
                        @click="loadMore"
                      >
                        加载更多
                      </button>
                    </div>
                  </div>
                </template>

                <!-- 收到的赞：列表 + 点赞详情（对齐主站） -->
                <template v-else-if="activeCat === 'like_aggregation'">
                  <div
                    v-if="likeDetailNotifId && likeDetailRow"
                    class="msg-like-detail-stack"
                  >
                    <nav
                      class="msg-main__title-card msg-like-detail__crumb-nav"
                      aria-label="面包屑"
                    >
                      <div class="msg-main__bar">
                        <button
                          type="button"
                          class="msg-like-detail__crumb-link"
                          @click="closeLikeDetail"
                        >
                          收到的赞
                        </button>
                        <span
                          class="msg-like-detail__crumb-sep"
                          aria-hidden="true"
                          >&nbsp;&gt;&nbsp;</span
                        >
                        <span class="msg-like-detail__crumb-here">点赞详情</span>
                      </div>
                    </nav>
                    <div
                      class="msg-like-detail__panel msg-like-detail__panel--ctx"
                    >
                      <button
                        type="button"
                        class="msg-like-detail__ctx-hit"
                        @click="onLikeCommentContextClick"
                      >
                        <span class="msg-like-detail__ctx-label"
                          >{{ likeDetailKindLabel }}:</span
                        >
                        <span class="msg-like-detail__ctx-body">{{
                          likeDetailContextBody
                        }}</span>
                      </button>
                    </div>
                    <div
                      class="msg-like-detail__panel msg-like-detail__panel--list"
                    >
                      <div
                        v-if="likeDetailLoading"
                        class="msg-like-detail__loading"
                      >
                        加载中…
                      </div>
                      <ul
                        v-else-if="likeDetailItems.length"
                        class="msg-like-detail__ul"
                        role="list"
                      >
                        <li
                          v-for="lk in likeDetailItems"
                          :key="lk.id"
                          class="msg-like-detail__liker"
                        >
                          <router-link
                            v-if="Number(lk.user_id)"
                            class="msg-like-detail__liker-face"
                            :to="likeSenderSpaceTo(lk.user_id)"
                            @click.stop
                          >
                            <img
                              :src="lk.avatar_url || defaultFace"
                              alt=""
                            />
                          </router-link>
                          <span v-else class="msg-like-detail__liker-face">
                            <img
                              :src="lk.avatar_url || defaultFace"
                              alt=""
                            />
                          </span>
                          <div class="msg-like-detail__liker-mid">
                            <div class="msg-like-detail__liker-line1">
                              <router-link
                                v-if="Number(lk.user_id)"
                                class="msg-like-detail__liker-name"
                                :to="likeSenderSpaceTo(lk.user_id)"
                                @click.stop
                                >{{ lk.username }}</router-link
                              >
                              <span v-else class="msg-like-detail__liker-name">{{
                                lk.username
                              }}</span>
                              <span class="msg-like-detail__liker-suffix"
                                >赞了我</span
                              >
                            </div>
                            <time
                              class="msg-like-detail__liker-time"
                              :datetime="lk.created_at"
                              >{{ formatTimeZh(lk.created_at) }}</time
                            >
                          </div>
                          <button
                            v-if="!likeLikerIsSelf(lk)"
                            type="button"
                            class="msg-like-detail__follow"
                            :class="{ 'is-followed': !!lk.followed_by_me }"
                            :disabled="
                              likeFollowPendingId === Number(lk.user_id)
                            "
                            @mouseenter="onLikeFollowHover(lk, true)"
                            @mouseleave="onLikeFollowHover(lk, false)"
                            @click.stop="onLikeFollowClick(lk)"
                          >
                            {{ likeFollowButtonLabel(lk) }}
                          </button>
                        </li>
                      </ul>
                      <p v-else class="msg-like-detail__empty">暂无点赞记录</p>
                      <div
                        v-if="likeDetailNextCursor && !likeDetailLoading"
                        class="msg-like-detail__more"
                      >
                        <button
                          type="button"
                          class="msg-reply-more__btn"
                          :disabled="likeDetailLoadingMore"
                          @click="loadLikeLikers(false)"
                        >
                          {{
                            likeDetailLoadingMore ? "加载中…" : "加载更多"
                          }}
                        </button>
                      </div>
                    </div>
                  </div>
                  <div v-else class="msg-reply-scroll">
                    <ul
                      v-if="likeRows.length"
                      class="msg-like-list"
                      role="list"
                    >
                      <li
                        v-for="row in likeRows"
                        :key="row.id"
                        class="msg-like-row"
                        :class="{ 'msg-like-row--unread': !row.is_read }"
                        @click="onLikeRowShellClick(row, $event)"
                      >
                        <div class="msg-like-row__cols">
                          <div
                            class="msg-like-row__faces"
                            :class="{
                              'msg-like-row__faces--single': !row.showSecondFace
                            }"
                            aria-hidden="true"
                          >
                            <router-link
                              v-if="row.userId1"
                              class="msg-like-row__face msg-like-row__face--first msg-like-row__face-link"
                              :to="likeSenderSpaceTo(row.userId1)"
                              @click.stop="onLikeSenderNav(row)"
                            >
                              <img
                                class="msg-like-row__face-img"
                                :src="row.face1"
                                alt=""
                              />
                            </router-link>
                            <img
                              v-else
                              class="msg-like-row__face msg-like-row__face--first"
                              :src="row.face1"
                              alt=""
                            />
                            <template v-if="row.showSecondFace">
                              <router-link
                                v-if="row.userId2"
                                class="msg-like-row__face msg-like-row__face--second msg-like-row__face-link"
                                :to="likeSenderSpaceTo(row.userId2)"
                                @click.stop="onLikeSenderNav(row)"
                              >
                                <img
                                  class="msg-like-row__face-img"
                                  :src="row.face2"
                                  alt=""
                                />
                              </router-link>
                              <img
                                v-else
                                class="msg-like-row__face msg-like-row__face--second"
                                :src="row.face2"
                                alt=""
                              />
                            </template>
                          </div>
                          <div class="msg-like-row__mid">
                            <p class="msg-like-row__line1">
                              <template v-if="row.total === 1">
                                <router-link
                                  v-if="row.userId1"
                                  class="msg-like-row__nick msg-like-row__nick-link"
                                  :to="likeSenderSpaceTo(row.userId1)"
                                  @click.stop="onLikeSenderNav(row)"
                                  >{{ row.name1 }}</router-link
                                >
                                <span v-else class="msg-like-row__nick">{{
                                  row.name1
                                }}</span>
                                <span class="msg-like-row__plain">
                                  赞了我的{{ row.targetZh }}</span
                                >
                              </template>
                              <template v-else-if="row.total === 2">
                                <router-link
                                  v-if="row.userId1"
                                  class="msg-like-row__nick msg-like-row__nick-link"
                                  :to="likeSenderSpaceTo(row.userId1)"
                                  @click.stop="onLikeSenderNav(row)"
                                  >{{ row.name1 }}</router-link
                                >
                                <span v-else class="msg-like-row__nick">{{
                                  row.name1
                                }}</span>
                                <span class="msg-like-row__plain">、</span>
                                <router-link
                                  v-if="row.userId2"
                                  class="msg-like-row__nick msg-like-row__nick-link"
                                  :to="likeSenderSpaceTo(row.userId2)"
                                  @click.stop="onLikeSenderNav(row)"
                                  >{{ row.name2 }}</router-link
                                >
                                <span v-else class="msg-like-row__nick">{{
                                  row.name2
                                }}</span>
                                <span class="msg-like-row__plain">
                                  赞了我的{{ row.targetZh }}</span
                                >
                              </template>
                              <template v-else>
                                <router-link
                                  v-if="row.userId1"
                                  class="msg-like-row__nick msg-like-row__nick-link"
                                  :to="likeSenderSpaceTo(row.userId1)"
                                  @click.stop="onLikeSenderNav(row)"
                                  >{{ row.name1 }}</router-link
                                >
                                <span v-else class="msg-like-row__nick">{{
                                  row.name1
                                }}</span>
                                <template v-if="row.name2">
                                  <span class="msg-like-row__plain">、</span>
                                  <router-link
                                    v-if="row.userId2"
                                    class="msg-like-row__nick msg-like-row__nick-link"
                                    :to="likeSenderSpaceTo(row.userId2)"
                                    @click.stop="onLikeSenderNav(row)"
                                    >{{ row.name2 }}</router-link
                                  >
                                  <span v-else class="msg-like-row__nick">{{
                                    row.name2
                                  }}</span>
                                </template>
                                <span class="msg-like-row__plain">
                                  等总计{{ row.total }}人赞了我的{{
                                    row.targetZh
                                  }}</span
                                >
                              </template>
                            </p>
                            <div class="msg-like-row__meta">
                              <time class="msg-like-row__time">{{
                                row.timeZh
                              }}</time>
                              <div class="msg-like-row__acts">
                                <button
                                  type="button"
                                  class="msg-like-row__act"
                                  @click.stop="onDeleteNotif(row)"
                                >
                                  <svg
                                    class="msg-like-row__act-ico"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="1.5"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    aria-hidden="true"
                                  >
                                    <path
                                      d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6h14zM10 11v6M14 11v6"
                                    />
                                  </svg>
                                  删除该通知
                                </button>
                                <button
                                  v-if="!row.likes_muted"
                                  type="button"
                                  class="msg-like-row__act"
                                  @click.stop="openMuteConfirm(row)"
                                >
                                  <svg
                                    class="msg-like-row__act-ico"
                                    viewBox="0 0 24 24"
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="1.5"
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    aria-hidden="true"
                                  >
                                    <circle cx="12" cy="12" r="9" />
                                    <path d="M5 5l14 14" />
                                  </svg>
                                  不再通知
                                </button>
                              </div>
                            </div>
                          </div>
                          <button
                            type="button"
                            class="msg-like-row__preview"
                            :class="{
                              'msg-like-row__preview--tap': row.isMediaLike
                            }"
                            :title="row.commentFull || row.preview || '—'"
                            @click.stop="
                              row.isMediaLike
                                ? onLikePreviewToContent(row)
                                : undefined
                            "
                          >
                            {{ row.previewDisplay }}
                          </button>
                        </div>
                      </li>
                    </ul>
                    <div v-else class="msg-no-data" aria-live="polite">
                      <div class="msg-no-data__panel">
                        <img
                          class="msg-no-data__img"
                          :src="emptyNoDataImg"
                          alt="然而并没有数据"
                        />
                      </div>
                    </div>
                    <div v-if="nextCursor && !listLoading" class="msg-reply-more">
                      <button
                        type="button"
                        class="msg-reply-more__btn"
                        @click="loadMore"
                      >
                        加载更多
                      </button>
                    </div>
                  </div>
                </template>

                <!-- 我的消息：会话列表 + 聊天窗 -->
                <MbDmChatPanel
                  v-else-if="activeCat === 'my_message'"
                  :peer-id-from-route="dmPeerIdFromRoute"
                />

                <!-- 其它分类：简要列表 -->
                <template v-else>
                  <div class="msg-generic-scroll">
                    <ul v-if="items.length" class="msg-generic-list">
                      <li
                        v-for="(n, i) in items"
                        :key="n.id ?? i"
                        class="msg-generic-row"
                      >
                        <p class="msg-generic-row__msg">
                          {{ n.message || "（无摘要）" }}
                        </p>
                        <p class="msg-generic-row__sub">
                          {{ n.comment_preview || "" }}
                        </p>
                        <time class="msg-generic-row__time">{{
                          formatTimeZh(n.created_at)
                        }}</time>
                      </li>
                    </ul>
                    <div v-else class="msg-no-data" aria-live="polite">
                      <div class="msg-no-data__panel">
                        <img
                          class="msg-no-data__img"
                          :src="emptyNoDataImg"
                          alt="然而并没有数据"
                        />
                      </div>
                    </div>
                  </div>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div
        v-if="deleteConfirmId"
        class="msg-del-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="msg-del-title"
      >
        <div
          class="msg-del-overlay__backdrop"
          aria-hidden="true"
          @click="closeDeleteConfirm"
        />
        <div class="msg-del-modal">
          <button
            type="button"
            class="msg-del-modal__close"
            aria-label="关闭"
            :disabled="deleteSubmitting"
            @click="closeDeleteConfirm"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              aria-hidden="true"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
          <h2 id="msg-del-title" class="msg-del-modal__title">删除</h2>
          <p class="msg-del-modal__desc">
            <template v-if="deleteConfirmKind === 'like'">
              该条通知删除后，当有新点赞时会重新出现在列表，是否继续？
            </template>
            <template v-else>
              删除该条通知后将无法恢复，是否继续？
            </template>
          </p>
          <div class="msg-del-modal__actions">
            <button
              type="button"
              class="msg-del-modal__btn msg-del-modal__btn--ghost"
              :disabled="deleteSubmitting"
              @click="closeDeleteConfirm"
            >
              取消
            </button>
            <button
              type="button"
              class="msg-del-modal__btn msg-del-modal__btn--primary"
              :disabled="deleteSubmitting"
              @click="confirmDeleteNotif"
            >
              确认
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="muteConfirmId"
        class="msg-del-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="msg-mute-title"
      >
        <div
          class="msg-del-overlay__backdrop"
          aria-hidden="true"
          @click="closeMuteConfirm"
        />
        <div class="msg-del-modal">
          <button
            type="button"
            class="msg-del-modal__close"
            aria-label="关闭"
            :disabled="muteSubmitting"
            @click="closeMuteConfirm"
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.75"
              stroke-linecap="round"
              aria-hidden="true"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
          <h2 id="msg-mute-title" class="msg-del-modal__title">不再通知</h2>
          <p class="msg-del-modal__desc">
            这条内容的点赞将不再通知，但仍可在列表内查看，是否继续？
          </p>
          <div class="msg-del-modal__actions">
            <button
              type="button"
              class="msg-del-modal__btn msg-del-modal__btn--ghost"
              :disabled="muteSubmitting"
              @click="closeMuteConfirm"
            >
              取消
            </button>
            <button
              type="button"
              class="msg-del-modal__btn msg-del-modal__btn--primary"
              :disabled="muteSubmitting"
              @click="confirmMuteNotif"
            >
              确认
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script>
import { ElMessage } from "element-plus";
import {
  mbListNotifications,
  mbListNotificationLikeLikers,
  mbMarkNotificationRead,
  mbMarkNotificationCategoryRead,
  mbMarkNotificationsReadBatch,
  mbDeleteNotification,
  mbMuteLikeNotification,
  mbToggleNotificationCommentLike,
  mbPostNotificationCommentReply,
  mbToggleUserFollow
} from "@/api/minibili";
import { getAccessToken } from "@/utils/authTokens";
import { openMinibiliLoginModal } from "@/utils/minibiliLoginModal";
import { MESSAGE_CATEGORIES, formatMessageUnreadBadge } from "@/utils/messageCategories";
import { refreshMessageUnread } from "@/utils/messageUnread";
import defaultFace from "@/assets/akari.jpg";
import {
  minibiliVideoPlayRoute,
  minibiliArticleReadRoute
} from "@/utils/minibiliRoutes";
import { formatVideoBvid } from "@/utils/videoBvid";
import MbDmChatPanel from "@/components/minibili/MbDmChatPanel.vue";
import emptyNoDataImg from "@/assets/empty_2.png";
import loadingTvImg from "@/assets/loading_tv.gif";

const SIDEBAR = MESSAGE_CATEGORIES;

/** SPEC F9：在消息中心列表内展示即视为已查看 */
const INBOX_AUTO_READ_CATS = new Set([
  "reply_received",
  "at_me",
  "like_aggregation",
  "system_notice"
]);

const REPLY_PLACEHOLDER =
  "请自觉遵守互联网相关的政策法规，严禁发布色情、暴力、反动的言论。";

const CAT_TO_TITLE = {
  my_message: "我的消息",
  reply_received: "回复我的",
  at_me: "@ 我的",
  like_aggregation: "收到的赞",
  system_notice: "系统通知"
};

function parseApiTime(s) {
  if (!s) return new Date();
  const m = /^(\d{4})-(\d{2})-(\d{2}) (\d{2}):(\d{2}):(\d{2})$/.exec(String(s));
  if (!m) return new Date(s);
  return new Date(
    Number(m[1]),
    Number(m[2]) - 1,
    Number(m[3]),
    Number(m[4]),
    Number(m[5]),
    Number(m[6])
  );
}

export default {
  name: "MinibiliMessages",
  components: { MbDmChatPanel },
  data() {
    return {
      msgPageBg: new URL("../../assets/light_bg.png@1c.webp", import.meta.url)
        .href,
      emptyNoDataImg,
      loadingTvImg,
      activeCat: "my_message",
      sidebarNav: SIDEBAR,
      msgUnread: {},
      items: [],
      nextCursor: "",
      listLoading: false,
      _backdropRestore: null,
      replyComposerNotifId: 0,
      replyComposerText: "",
      replySubmitting: false,
      replyPlaceholder: REPLY_PLACEHOLDER,
      deleteConfirmId: 0,
      deleteConfirmKind: "reply",
      deleteSubmitting: false,
      muteConfirmId: 0,
      muteSubmitting: false,
      likeDetailNotifId: 0,
      likeDetailItems: [],
      likeDetailNextCursor: "",
      likeDetailLoading: false,
      likeDetailLoadingMore: false,
      likeFollowPendingId: 0,
      likeFollowHoverId: 0
    };
  },
  computed: {
    token() {
      return getAccessToken();
    },
    mainTitle() {
      return CAT_TO_TITLE[this.activeCat] || "消息中心";
    },
    dmPeerIdFromRoute() {
      const raw = this.$route.query.peer_id;
      const n = parseInt(String(raw || ""), 10);
      return Number.isFinite(n) && n > 0 ? n : 0;
    },
    meReplyFace() {
      const me = this.$store.state.login.minibiliMe;
      const u = me && String(me.avatar_url || "").trim();
      if (u) return u;
      const p = this.$store.state.login.proInfo;
      if (p && typeof p === "object" && !Array.isArray(p)) {
        const f = String(p.face || "").trim();
        if (f) return f;
      }
      return defaultFace;
    },
    meUserId() {
      const me = this.$store.state.login.minibiliMe;
      const id = me && Number(me.user_id);
      return Number.isFinite(id) && id > 0 ? id : 0;
    },
    likeRows() {
      if (this.activeCat !== "like_aggregation") return [];
      return (this.items || []).map(n => {
        const names = Array.isArray(n.sender_names) ? n.sender_names : [];
        const rawAvatars = Array.isArray(n.sender_avatar_urls)
          ? n.sender_avatar_urls
          : [];
        const pickFace = i => {
          const u = String(rawAvatars[i] || "").trim();
          return u || defaultFace;
        };
        const total = Number(n.total_likes) || 0;
        const name1 = String(names[0] || "用户").trim() || "用户";
        const name2Raw = names.length > 1 ? String(names[1] || "").trim() : "";
        const name2 = name2Raw || "";
        const rawIds = Array.isArray(n.sender_user_ids) ? n.sender_user_ids : [];
        const userId1 = Number(rawIds[0]) || 0;
        const userId2 = Number(rawIds[1]) || 0;
        const showSecondFace = total >= 2;
        const vid = Number(n.video_id) || 0;
        const aid = Number(n.article_id) || 0;
        const likedCid = Number(n.liked_comment_id) || 0;
        const commentFull = String(n.comment_full_text || "").trim();
        const targetZh = String(n.like_target || "评论").trim() || "评论";
        const isDanmaku = targetZh === "弹幕";
        const isArticleComment = !isDanmaku && aid > 0;
        const isVideoComment = !isDanmaku && !isArticleComment && vid > 0;
        const isMediaLike = isArticleComment || isVideoComment;
        const coverUrl = isArticleComment
          ? String(n.article_cover_url || "").trim()
          : "";
        const previewRaw = String(n.comment_preview || "").trim();
        const previewSource = previewRaw || commentFull;
        return {
          id: Number(n.id) || 0,
          total,
          name1,
          name2,
          userId1,
          userId2,
          targetZh,
          preview: previewRaw,
          previewDisplay: this.msgReplyPreview15(previewSource || "—"),
          commentFull: commentFull || previewRaw,
          timeZh: this.formatTimeZh(n.created_at),
          is_read: !!n.is_read,
          showSecondFace,
          face1: pickFace(0),
          face2: pickFace(1),
          likes_muted: !!n.likes_muted,
          isDanmaku,
          isArticleComment,
          isVideoComment,
          isMediaLike,
          coverUrl,
          video_id: vid,
          article_id: aid,
          liked_comment_id: likedCid,
          reply_comment_id: likedCid
        };
      });
    },
    replyRows() {
      if (this.activeCat !== "reply_received") return [];
      return (this.items || []).map(n => {
        const name = String(n.sender_username || "用户").trim() || "用户";
        const face = String(n.sender_avatar_url || "").trim() || defaultFace;
        const replyText = String(n.reply_content || "").trim() || "（无内容）";
        const t = String(n.type || "");
        let kind = String(n.inbox_kind || "").trim();
        if (!kind) {
          if (t === "video_comment_received") {
            kind = "video_comment";
          } else if (t === "article_comment_received") {
            kind = "article_comment";
          } else if (t === "article_reply_received") {
            kind = "article_reply";
          } else {
            kind = "reply_to_comment";
          }
        }
        // type 优先：避免 inbox_kind 与通知类型不一致时走错点赞/回复接口
        if (t === "video_comment_received") {
          kind = "video_comment";
        } else if (t === "article_comment_received") {
          kind = "article_comment";
        } else if (t === "article_reply_received") {
          kind = "article_reply";
        }
        const isVideoComment = kind === "video_comment";
        const isArticleComment = kind === "article_comment";
        const isArticleReply = kind === "article_reply";
        const isMediaComment = isVideoComment || isArticleComment;
        const coverUrl = isVideoComment
          ? String(n.video_cover_url || "").trim()
          : isArticleComment
            ? String(n.article_cover_url || "").trim()
            : "";
        const preview = !isMediaComment
          ? String(
              n.parent_content_preview || n.comment_preview || ""
            ).trim()
          : "";
        const suffix = isVideoComment
          ? "对我的视频发表了评论"
          : isArticleComment
            ? "对我的文章发表了评论"
            : "回复了我的评论";
        const previewRaw =
          preview || (isMediaComment ? "" : "—");
        return {
          id: Number(n.id) || 0,
          name,
          face,
          replyText,
          replyTextDisplay: this.msgReplyPreview15(replyText),
          suffix,
          isVideoComment,
          isArticleComment,
          isArticleReply,
          isMediaComment,
          preview: previewRaw,
          previewDisplay: this.msgReplyPreview15(previewRaw),
          coverUrl,
          timeZh: this.formatTimeZh(n.created_at),
          is_read: !!n.is_read,
          video_id: Number(n.video_id) || 0,
          article_id: Number(n.article_id) || 0,
          reply_comment_id: Number(n.reply_comment_id) || 0,
          liked_by_me: !!n.liked_by_me
        };
      });
    },
    msgBgLayerStyle() {
      return { backgroundImage: `url(${this.msgPageBg})` };
    },
    /** 点赞详情：当前选中的列表行 */
    likeDetailRow() {
      const id = Number(this.likeDetailNotifId) || 0;
      if (!id) return null;
      return this.likeRows.find(r => Number(r.id) === id) || null;
    },
    isLikeDetailView() {
      return (
        this.activeCat === "like_aggregation" &&
        !!this.likeDetailNotifId &&
        !!this.likeDetailRow
      );
    },
    likeDetailKindLabel() {
      const r = this.likeDetailRow;
      if (!r) return "评论";
      if (r.targetZh === "弹幕") return "弹幕";
      if (r.isArticleComment) return "专栏评论";
      return "评论";
    },
    likeDetailContextBody() {
      const r = this.likeDetailRow;
      if (!r) return "—";
      const t = String(r.commentFull || r.preview || "").trim();
      return t || "—";
    },
    /** 当前 Tab 是否已有列表数据（区分首屏加载与「加载更多」） */
    hasMessagesListContent() {
      if (this.activeCat === "my_message") return true;
      if (this.activeCat === "reply_received") return this.replyRows.length > 0;
      if (this.activeCat === "like_aggregation") return this.likeRows.length > 0;
      return (this.items || []).length > 0;
    },
    /** 首屏拉取中：隐藏大白框，仅标题下窄白条 + TV（loadMore 不触发） */
    showMsgCenterPageLoading() {
      return (
        this.listLoading &&
        this.activeCat !== "my_message" &&
        !this.hasMessagesListContent
      );
    },
    /** 有列表时保留大白底内容框；无消息时仅保留空态图框，与标题条同宽；首屏加载中透明底 */
    showMessagesBodyChrome() {
      if (this.showMsgCenterPageLoading) return false;
      if (this.activeCat === "my_message") return true;
      if (this.activeCat === "reply_received") {
        return this.replyRows.length > 0;
      }
      if (this.activeCat === "like_aggregation") {
        return this.likeRows.length > 0;
      }
      return (this.items || []).length > 0;
    }
  },
  watch: {
    token(v) {
      if (v) this.bootstrap();
    },
    activeCat(v) {
      if (v !== "like_aggregation") {
        this.closeLikeDetail();
      }
      if (this.token) void this.loadList();
    },
    "$route.query.cat"(v) {
      this.syncCatFromRoute(v);
    },
    "$route.query.like_detail"(raw) {
      if (this.activeCat !== "like_aggregation") return;
      const id = Number(raw) || 0;
      if (!id) {
        if (this.likeDetailNotifId) {
          this.closeLikeDetail({ skipRouter: true });
        }
        return;
      }
      if (Number(this.likeDetailNotifId) === id) return;
      const row = this.likeRows.find(r => Number(r.id) === id);
      this.likeDetailNotifId = id;
      this.likeDetailItems = [];
      this.likeDetailNextCursor = "";
      void this.loadLikeLikers(true);
      if (row) void this.onLikeRowClick(row);
    },
    deleteConfirmId() {
      this.syncMessageModalEsc();
    },
    muteConfirmId() {
      this.syncMessageModalEsc();
    }
  },
  mounted() {
    this.applyMsgPageBackdrop();
    if (!this.$route.query.cat) {
      void this.$router.replace({
        path: "/minibili/messages",
        query: { ...this.$route.query, cat: "my_message" }
      });
    }
    this.syncCatFromRoute(this.$route.query.cat);
    if (this.token) void this.bootstrap();
  },
  activated() {
    this.applyMsgPageBackdrop();
  },
  deactivated() {
    this.clearMsgPageBackdrop();
  },
  beforeUnmount() {
    this.clearMsgPageBackdrop();
    if (this._msgModalEscHandler) {
      document.removeEventListener("keydown", this._msgModalEscHandler);
      this._msgModalEscHandler = null;
    }
  },
  methods: {
    openLoginModal() {
      openMinibiliLoginModal({
        tab: 0,
        redirect: "/minibili/messages?cat=my_message"
      });
    },
    async fetchMsgUnread() {
      const summary = await refreshMessageUnread();
      this.msgUnread = summary || {};
    },
    formatTimeZh(createdAt) {
      const d = parseApiTime(createdAt);
      const y = d.getFullYear();
      const mo = d.getMonth() + 1;
      const day = d.getDate();
      const h = String(d.getHours()).padStart(2, "0");
      const mi = String(d.getMinutes()).padStart(2, "0");
      return `${y}年${mo}月${day}日 ${h}:${mi}`;
    },
    syncCatFromRoute(cat) {
      const c = String(cat || "");
      const ok = SIDEBAR.some(s => s.cat === c);
      this.activeCat = ok ? c : "my_message";
    },
    formatMessageUnreadBadge,
    onSidebarClick(cat) {
      this.$router.replace({ path: "/minibili/messages", query: { cat } });
    },
    applyMsgPageBackdrop() {
      const html = document.documentElement;
      const body = document.body;
      const app = document.getElementById("app");
      const appBody = document.querySelector(".app-body");
      if (!this._backdropRestore) {
        this._backdropRestore = {
          html: html.style.backgroundColor,
          body: body.style.backgroundColor,
          app: app ? app.style.backgroundColor : "",
          appBody: appBody ? appBody.style.backgroundColor : ""
        };
      }
      const c = "#dff6f0";
      html.style.backgroundColor = c;
      body.style.backgroundColor = c;
      if (app) app.style.backgroundColor = c;
      if (appBody) appBody.style.backgroundColor = c;
    },
    clearMsgPageBackdrop() {
      const r = this._backdropRestore;
      if (!r) return;
      const html = document.documentElement;
      const body = document.body;
      const app = document.getElementById("app");
      const appBody = document.querySelector(".app-body");
      html.style.backgroundColor = r.html;
      body.style.backgroundColor = r.body;
      if (app) app.style.backgroundColor = r.app;
      if (appBody) appBody.style.backgroundColor = r.appBody;
      this._backdropRestore = null;
    },
    async bootstrap() {
      await Promise.all([this.loadList(), this.fetchMsgUnread()]);
    },
    async loadList() {
      if (!this.token) return;
      if (this.activeCat === "my_message") {
        return;
      }
      this.listLoading = true;
      this.items = [];
      this.nextCursor = "";
      this.replyComposerNotifId = 0;
      this.replyComposerText = "";
      try {
        const { items, next_cursor: next } = await mbListNotifications({
          category: this.activeCat
        });
        this.items = items || [];
        this.nextCursor = next || "";
        await this.markVisibleNotificationsRead();
      } catch (e) {
        ElMessage.error((e && e.message) || "加载通知失败");
      } finally {
        this.listLoading = false;
        if (this.activeCat === "like_aggregation") {
          this.syncLikeDetailFromRoute();
        }
      }
    },
    async loadMore() {
      if (!this.nextCursor || this.listLoading) return;
      this.listLoading = true;
      try {
        const { items, next_cursor: next } = await mbListNotifications({
          category: this.activeCat,
          cursor: this.nextCursor
        });
        const more = items || [];
        this.items = [...this.items, ...more];
        this.nextCursor = next || "";
        await this.markVisibleNotificationsRead(more);
      } catch (e) {
        ElMessage.error((e && e.message) || "加载失败");
      } finally {
        this.listLoading = false;
      }
    },
    /** 当前分类内通知标为已读（SPEC F9） */
    async markVisibleNotificationsRead(onlyItems) {
      if (!INBOX_AUTO_READ_CATS.has(this.activeCat)) return;
      if (this.activeCat === "like_aggregation") {
        try {
          await mbMarkNotificationCategoryRead("like_aggregation");
          this.items = (this.items || []).map(n => ({ ...n, is_read: true }));
          void this.fetchMsgUnread();
        } catch {
          /* 已读失败不阻断列表展示 */
        }
        return;
      }
      const pool = Array.isArray(onlyItems) ? onlyItems : this.items || [];
      const ids = pool
        .filter(n => !n.is_read)
        .map(n => Number(n.id) || 0)
        .filter(id => id > 0);
      if (!ids.length) return;
      try {
        await mbMarkNotificationsReadBatch(ids);
        const idSet = new Set(ids);
        this.items = (this.items || []).map(n =>
          idSet.has(Number(n.id)) ? { ...n, is_read: true } : n
        );
        void this.fetchMsgUnread();
      } catch {
        /* 已读失败不阻断列表展示 */
      }
    },
    async onReplyRowClick(row) {
      if (!row.id) return;
      try {
        await mbMarkNotificationRead(row.id);
        const ix = this.items.findIndex(x => Number(x.id) === row.id);
        if (ix >= 0) this.items.splice(ix, 1, { ...this.items[ix], is_read: true });
        void this.fetchMsgUnread();
      } catch {
        /* 忽略已读失败 */
      }
    },
    /** 点击通知栏非功能区域：新开视频页并定位评论（与 pushVideoWithComment 一致） */
    onReplyRowShellClick(row, e) {
      if (!row || !row.id) return;
      const el = e && e.target;
      if (!(el instanceof Node)) return;
      if (el.closest("button, textarea, a[href], input, select, .msg-reply-compose")) {
        return;
      }
      if (row.isVideoComment) {
        this.pushVideoWithComment(row);
      } else if (row.isArticleComment) {
        this.pushArticleWithComment(row);
      } else {
        void this.onReplyRowClick(row);
      }
    },
    /** 发送者头像 / 昵称：预留个人空间入口（后续接路由） */
    onReplySenderProfile(_row) {
      /* TODO: 跳转 Mini-Bili 个人空间 /up/{uid}，需通知接口带 sender_user_id */
    },
    likeSenderSpaceTo(userId) {
      const id = Number(userId) || 0;
      return {
        name: "minibiliUserSpace",
        params: { userId: String(id) }
      };
    },
    onLikeSenderNav(row) {
      void this.onLikeRowClick(row);
    },
    closeLikeDetail(options) {
      const skipRouter = options && options.skipRouter === true;
      this.likeDetailNotifId = 0;
      this.likeDetailItems = [];
      this.likeDetailNextCursor = "";
      this.likeDetailLoading = false;
      this.likeDetailLoadingMore = false;
      this.likeFollowPendingId = 0;
      this.likeFollowHoverId = 0;
      if (
        !skipRouter &&
        this.$route.query.like_detail != null &&
        String(this.$route.query.like_detail) !== ""
      ) {
        const q = { ...this.$route.query };
        delete q.like_detail;
        void this.$router.replace({
          path: this.$route.path,
          query: q
        });
      }
    },
    /** 刷新列表后：若地址栏带 like_detail，恢复点赞详情态 */
    syncLikeDetailFromRoute() {
      if (this.activeCat !== "like_aggregation") return;
      const id = Number(this.$route.query.like_detail) || 0;
      if (!id) return;
      if (Number(this.likeDetailNotifId) === id && this.likeDetailRow) return;
      const row = this.likeRows.find(r => Number(r.id) === id);
      if (!row) return;
      this.likeDetailNotifId = id;
      this.likeDetailItems = [];
      this.likeDetailNextCursor = "";
      void this.loadLikeLikers(true);
      void this.onLikeRowClick(row);
    },
    openLikeDetail(row) {
      const id = Number(row && row.id) || 0;
      if (!id) return;
      this.likeDetailNotifId = id;
      this.likeDetailItems = [];
      this.likeDetailNextCursor = "";
      void this.loadLikeLikers(true);
      void this.onLikeRowClick(row);
      if (String(this.$route.query.like_detail) !== String(id)) {
        void this.$router.push({
          path: this.$route.path,
          query: { ...this.$route.query, cat: "like_aggregation", like_detail: String(id) }
        });
      }
    },
    async loadLikeLikers(isReset) {
      const nid = Number(this.likeDetailNotifId) || 0;
      if (!nid) return;
      if (isReset) {
        this.likeDetailLoading = true;
        this.likeDetailItems = [];
        this.likeDetailNextCursor = "";
      } else {
        this.likeDetailLoadingMore = true;
      }
      try {
        const cur =
          !isReset && this.likeDetailNextCursor
            ? { cursor: this.likeDetailNextCursor }
            : undefined;
        const { items, next_cursor: next } = await mbListNotificationLikeLikers(
          nid,
          cur
        );
        const list = items || [];
        this.likeDetailItems = isReset
          ? list
          : [...this.likeDetailItems, ...list];
        this.likeDetailNextCursor = next || "";
      } catch (e) {
        ElMessage.error((e && e.message) || "加载失败");
      } finally {
        this.likeDetailLoading = false;
        this.likeDetailLoadingMore = false;
      }
    },
    onLikeRowShellClick(row, e) {
      if (!row || !row.id) return;
      const el = e && e.target;
      if (!(el instanceof Node)) return;
      if (
        el.closest(
          "button, a[href], .msg-like-row__face-link, .msg-like-row__nick-link"
        )
      ) {
        return;
      }
      this.openLikeDetail(row);
    },
    onLikePreviewToContent(row) {
      void this.onLikeRowClick(row);
      this.pushLikeCommentTarget(row);
    },
    pushLikeCommentTarget(row) {
      if (!row) return;
      if (row.isArticleComment) {
        this.pushArticleWithComment(row);
        return;
      }
      if (Number(row.video_id) > 0) {
        this.pushVideoWithComment(row);
      }
    },
    onLikeCommentContextClick() {
      const row = this.likeDetailRow;
      if (!row) return;
      void this.onLikeRowClick(row);
      this.pushLikeCommentTarget(row);
    },
    likeLikerIsSelf(lk) {
      const uid = Number(lk && lk.user_id) || 0;
      return uid > 0 && uid === this.meUserId;
    },
    likeFollowButtonLabel(lk) {
      const uid = Number(lk && lk.user_id) || 0;
      if (this.likeFollowPendingId === uid) return "…";
      if (!lk || !lk.followed_by_me) return "+ 关注";
      if (this.likeFollowHoverId === uid) return "取消关注";
      return "已关注";
    },
    onLikeFollowHover(lk, enter) {
      const uid = Number(lk && lk.user_id) || 0;
      if (!uid || !lk || !lk.followed_by_me) {
        if (!enter) this.likeFollowHoverId = 0;
        return;
      }
      this.likeFollowHoverId = enter ? uid : 0;
    },
    async onLikeFollowClick(lk) {
      const uid = Number(lk && lk.user_id) || 0;
      if (!uid || this.likeLikerIsSelf(lk)) return;
      if (!getAccessToken()) {
        openMinibiliLoginModal({ tab: 0 });
        return;
      }
      if (this.likeFollowPendingId === uid) return;
      this.likeFollowPendingId = uid;
      try {
        const res = await mbToggleUserFollow(uid);
        const followed = !!res.followed;
        const ix = this.likeDetailItems.findIndex(
          x => Number(x.user_id) === uid
        );
        if (ix >= 0) {
          this.likeDetailItems.splice(ix, 1, {
            ...this.likeDetailItems[ix],
            followed_by_me: followed
          });
        }
        if (!followed && this.likeFollowHoverId === uid) {
          this.likeFollowHoverId = 0;
        }
      } catch (e) {
        ElMessage.error((e && e.message) || "关注失败");
      } finally {
        if (this.likeFollowPendingId === uid) {
          this.likeFollowPendingId = 0;
        }
      }
    },
    onLikeRowClick(row) {
      void this.onReplyRowClick(row);
    },
    onToggleReplyComposer(row) {
      const id = Number(row.id) || 0;
      if (!id) return;
      if (this.replyComposerNotifId === id) {
        this.replyComposerNotifId = 0;
        this.replyComposerText = "";
        return;
      }
      this.replyComposerNotifId = id;
      this.replyComposerText = "";
      if (!row.is_read) void this.onReplyRowClick(row);
    },
    onGoToVideoComment(row) {
      if (!row.isVideoComment) return;
      this.pushVideoWithComment(row);
    },
    onGoToArticleComment(row) {
      if (!row.isArticleComment) return;
      this.pushArticleWithComment(row);
    },
    pushVideoWithComment(row) {
      void this.onReplyRowClick(row);
      const vid = row.video_id;
      if (!vid) {
        ElMessage.warning("缺少视频信息");
        return;
      }
      const cid = Number(row.reply_comment_id) || 0;
      const query = cid > 0 ? { mb_cid: String(cid) } : {};
      const resolved = this.$router.resolve({
        ...minibiliVideoPlayRoute(vid),
        query
      });
      const href = resolved.href;
      const bvid = formatVideoBvid(vid);
      const url =
        href && /^https?:\/\//i.test(href)
          ? href
          : new URL(
              href || `#/video/${encodeURIComponent(bvid || String(vid))}`,
              window.location.href
            ).href;
      window.open(url, "_blank", "noopener,noreferrer");
    },
    pushArticleWithComment(row) {
      void this.onReplyRowClick(row);
      const aid = Number(row.article_id) || 0;
      if (!aid) {
        ElMessage.warning("缺少专栏信息");
        return;
      }
      const cid = Number(row.reply_comment_id) || 0;
      const query = cid > 0 ? { mb_cid: String(cid) } : {};
      const base = minibiliArticleReadRoute(aid);
      const resolved = this.$router.resolve(
        base ? { ...base, query } : { path: `/minibili/article/${aid}`, query }
      );
      const href = resolved.href;
      const url =
        href && /^https?:\/\//i.test(href)
          ? href
          : new URL(
              href || `#/minibili/article/${aid}`,
              window.location.href
            ).href;
      window.open(url, "_blank", "noopener,noreferrer");
    },
    apiErrMsg(e, fallback) {
      const data = e && e.response && e.response.data;
      if (data && data.code === 40400) {
        return "原评论已不存在，可删除该通知";
      }
      const msg = data && typeof data.msg === "string" ? data.msg : "";
      return msg || (e && e.message) || fallback;
    },
    async onLikeReply(row) {
      const nid = Number(row.id) || 0;
      if (!nid) {
        ElMessage.warning("缺少通知信息");
        return;
      }
      try {
        const { liked } = await mbToggleNotificationCommentLike(nid);
        const ix = this.items.findIndex(x => Number(x.id) === nid);
        if (ix >= 0) {
          this.items.splice(ix, 1, {
            ...this.items[ix],
            liked_by_me: liked
          });
        }
      } catch (e) {
        ElMessage.error(this.apiErrMsg(e, "点赞失败"));
      }
    },
    /**
     * 「回复我的」列表：评论/预览文案最多展示前 15 个字符（SPEC F9 / Skill S-010 与 replyCharCount 一致按 Unicode 码位计）。
     */
    msgReplyPreview15(text) {
      const raw = String(text ?? "");
      const chars = Array.from(raw);
      if (chars.length <= 15) return raw;
      return `${chars.slice(0, 15).join("")}…`;
    },
    replyCharCount(text) {
      return Array.from(String(text || "")).length;
    },
    async submitInlineReply(row) {
      const nid = Number(row.id) || 0;
      if (!nid) {
        ElMessage.warning("缺少通知信息");
        return;
      }
      const raw = String(this.replyComposerText || "");
      const content = raw.trim();
      const n = this.replyCharCount(content);
      if (n < 1 || n > 1000) {
        ElMessage.warning("评论长度为 1～1000 字");
        return;
      }
      this.replySubmitting = true;
      try {
        await mbPostNotificationCommentReply(nid, content);
        ElMessage({ type: "success", message: "发表成功", duration: 1600 });
        this.replyComposerNotifId = 0;
        this.replyComposerText = "";
      } catch (e) {
        ElMessage.error(this.apiErrMsg(e, "发表失败"));
      } finally {
        this.replySubmitting = false;
      }
    },
    onDeleteNotif(row) {
      const id = Number(row.id) || 0;
      if (!id) return;
      this.deleteConfirmId = id;
      this.deleteConfirmKind =
        this.activeCat === "like_aggregation" ? "like" : "reply";
    },
    syncMessageModalEsc() {
      if (this._msgModalEscHandler) {
        document.removeEventListener("keydown", this._msgModalEscHandler);
        this._msgModalEscHandler = null;
      }
      if (!this.deleteConfirmId && !this.muteConfirmId) return;
      this._msgModalEscHandler = e => {
        if (e.key !== "Escape") return;
        if (this.muteConfirmId) this.closeMuteConfirm();
        else if (this.deleteConfirmId) this.closeDeleteConfirm();
      };
      document.addEventListener("keydown", this._msgModalEscHandler);
    },
    closeDeleteConfirm() {
      if (this.deleteSubmitting) return;
      this.deleteConfirmId = 0;
      this.deleteConfirmKind = "reply";
    },
    closeMuteConfirm() {
      if (this.muteSubmitting) return;
      this.muteConfirmId = 0;
    },
    openMuteConfirm(row) {
      const id = Number(row.id) || 0;
      if (!id) return;
      this.muteConfirmId = id;
    },
    async confirmMuteNotif() {
      const id = Number(this.muteConfirmId) || 0;
      if (!id || this.muteSubmitting) return;
      this.muteSubmitting = true;
      try {
        await mbMuteLikeNotification(id);
        const ix = this.items.findIndex(x => Number(x.id) === id);
        if (ix >= 0) {
          this.items.splice(ix, 1, {
            ...this.items[ix],
            likes_muted: true
          });
        }
        this.muteConfirmId = 0;
        ElMessage({ type: "success", message: "已关闭点赞通知", duration: 1600 });
      } catch (e) {
        ElMessage.error((e && e.message) || "操作失败");
      } finally {
        this.muteSubmitting = false;
      }
    },
    async confirmDeleteNotif() {
      const id = Number(this.deleteConfirmId) || 0;
      if (!id || this.deleteSubmitting) return;
      this.deleteSubmitting = true;
      try {
        await mbDeleteNotification(id);
        if (Number(this.replyComposerNotifId) === id) {
          this.replyComposerNotifId = 0;
          this.replyComposerText = "";
        }
        if (Number(this.likeDetailNotifId) === id) {
          this.closeLikeDetail();
        }
        this.items = this.items.filter(x => Number(x.id) !== id);
        this.deleteConfirmId = 0;
        this.deleteConfirmKind = "reply";
        ElMessage({ type: "success", message: "已删除", duration: 1600 });
        void this.fetchMsgUnread();
      } catch (e) {
        ElMessage.error((e && e.message) || "删除失败");
      } finally {
        this.deleteSubmitting = false;
      }
    }
  }
};
</script>

<style scoped lang="scss">
$c-blue: #00aeec;
$c-text: #18191c;
$c-sub: #9499a0;
$c-line: #e3e5e7;
$c-row-on: #f4f5f7;

.msg-page {
  position: relative;
  isolation: isolate;
  min-height: 100vh;
  box-sizing: border-box;
}

.msg-bg {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 0;
  pointer-events: none;
  background-color: transparent;
  background-repeat: no-repeat;
  background-size: cover;
  background-position: center bottom;
}

.msg-root {
  position: relative;
  z-index: 1;
  box-sizing: border-box;
  min-height: 100vh;
  width: 100%;
  padding: 20px 16px 24px;
  background: transparent;
}

.msg-login-hint {
  display: flex;
  justify-content: center;
  padding-top: 48px;
}

.msg-login-card {
  max-width: 480px;
  padding: 28px 32px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 4px 12px rgba(0, 0, 0, 0.06);
  font-size: 14px;
  color: $c-text;
  a {
    color: $c-blue;
  }
}

.msg-login-card__crumbs {
  margin: 0 0 12px;
  font-size: 13px;
}

.msg-login-card__link {
  color: #00aeec;
  text-decoration: none;
  cursor: pointer;
  &:hover {
    color: #00b5e5;
  }
}

.msg-aside__crumb {
  color: #61666d;
  text-decoration: none;
  &:hover {
    color: #00aeec;
  }
}

.msg-aside__crumb-sep {
  color: #c9ccd0;
  user-select: none;
  margin: 0 4px;
}

.msg-layout {
  max-width: 1160px;
  margin: 0 auto;
}

.msg-stack {
  width: 100%;
}

.msg-panel {
  display: flex;
  align-items: stretch;
  height: calc(100vh - 88px);
  max-height: calc(100vh - 88px);
  min-height: calc(100vh - 88px);
  background: rgba(255, 255, 255, 0.24);
  -webkit-backdrop-filter: blur(18px);
  backdrop-filter: blur(18px);
  border: 1px solid rgba(255, 255, 255, 0.42);
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 105, 128, 0.08);
  overflow: hidden;
}

.msg-aside {
  width: 168px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 20px 0 16px;
  box-sizing: border-box;
  /* 左侧更透，但仍比主栏略「实」一点（alpha 更高） */
  background: rgba(255, 255, 255, 0.46);
  -webkit-backdrop-filter: blur(10px);
  backdrop-filter: blur(10px);
  border-right: 1px solid rgba(0, 0, 0, 0.06);
}

.msg-aside__head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 18px 20px;
  font-size: 16px;
  font-weight: 600;
  color: $c-text;
}

.msg-aside__plane {
  display: flex;
  flex-shrink: 0;
}

.msg-aside__head-text {
  line-height: 1.2;
}

.msg-aside__nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 0 10px;
  min-height: 0;
}

.msg-aside__link {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  width: 100%;
  padding: 11px 10px 11px 22px;
  border: none;
  background: transparent;
  text-align: left;
  font-size: 14px;
  color: #61666d;
  cursor: pointer;
  border-radius: 4px;
  line-height: 20px;
  &::before {
    content: "";
    position: absolute;
    left: 8px;
    top: 50%;
    transform: translateY(-50%);
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: #b7bdc4;
  }
  &:hover {
    background: rgba(0, 174, 236, 0.06);
  }
}

.msg-aside__link-label {
  flex: 1;
  min-width: 0;
}

.msg-aside__badge {
  flex-shrink: 0;
  min-width: 18px;
  height: 16px;
  padding: 0 5px;
  border-radius: 8px;
  background: #fb7299;
  color: #fff;
  font-size: 11px;
  line-height: 16px;
  text-align: center;
  box-sizing: border-box;
}

.msg-aside__link--on {
  color: $c-blue;
  font-weight: 500;
  &::before {
    background: $c-blue;
  }
}

.msg-aside__link--settings {
  gap: 6px;
  margin-top: 16px;
  padding-left: 12px;
  color: #61666d;
  &::before {
    display: none;
    content: none;
  }
  &:hover {
    color: $c-blue;
  }
}

.msg-aside__gear {
  display: flex;
  flex-shrink: 0;
}

.msg-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-sizing: border-box;
  /* 主栏：更透一些，与左侧同系毛玻璃 */
  padding: 16px;
  gap: 16px;
  background: rgba(255, 255, 255, 0.26);
  -webkit-backdrop-filter: blur(14px);
  backdrop-filter: blur(14px);
}

.msg-main__title-card {
  flex-shrink: 0;
  background: #fff;
  border: 1px solid $c-line;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.msg-main__body-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
  margin: 0;
  border-radius: 8px;
  border: 1px solid $c-line;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
}

.msg-main__body-card--bare {
  background: transparent;
  border: none;
  box-shadow: none;
}

.msg-main__body-card--bare .msg-generic-scroll {
  background: transparent;
  padding-left: 0;
  padding-right: 0;
}

.msg-main__body-card--bare .msg-reply-scroll {
  background: transparent;
}

.msg-main__body-card--bare .msg-like-detail-stack {
  padding: 0;
  background: transparent;
}

.msg-main__body-card--bare .msg-no-data {
  padding: 0;
}

.msg-main__body-card--bare .msg-no-data__panel {
  width: 100%;
}

.msg-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.msg-loading-strip {
  flex: 0 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.msg-loading-strip__card {
  width: 100%;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 22px 24px 24px;
  border: 1px solid $c-line;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.msg-loading-strip__tv {
  display: block;
  width: 64px;
  height: auto;
  max-width: 100%;
}

.msg-main__bar {
  display: flex;
  align-items: center;
  min-height: 52px;
  padding: 14px 20px;
  box-sizing: border-box;
}

.msg-main__title {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: #61666d;
  line-height: 24px;
}

/* —— 回复我的列表 —— */
.msg-reply-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  background: #fff;
  display: flex;
  flex-direction: column;
}

/* 无数据：内层白框与上方标题条同宽，略高于图，四周留白与参考图一致 */
.msg-no-data {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: flex-start;
  min-height: 0;
  width: 100%;
  box-sizing: border-box;
  padding: 16px 20px 20px;
  background: transparent;
}

.msg-no-data__panel {
  width: 100%;
  max-width: none;
  margin: 0;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 288px;
  padding: 36px 24px 40px;
  border: 1px solid $c-line;
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.msg-no-data__img {
  display: block;
  width: 240px;
  max-width: min(240px, 100%);
  height: auto;
}

.msg-reply-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.msg-reply-row {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0;
  padding: 20px 20px 18px;
  border-bottom: 1px solid $c-line;
  box-sizing: border-box;
  cursor: pointer;
  &:last-child {
    border-bottom: none;
  }
}

.msg-reply-row__cols {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  width: 100%;
  min-width: 0;
}

.msg-reply-row--unread {
  background: #fafbfd;
}

.msg-reply-row__face {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  overflow: hidden;
  background: #f1f2f3;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
}

.msg-reply-row__face--tap {
  display: block;
  margin: 0;
  padding: 0;
  border: none;
  cursor: pointer;
  font: inherit;
  color: inherit;
  text-align: left;
  &:focus-visible {
    outline: 2px solid rgba(0, 174, 236, 0.55);
    outline-offset: 2px;
  }
  &:hover img {
    opacity: 0.92;
  }
}

.msg-reply-row__mid {
  flex: 1;
  min-width: 0;
}

.msg-reply-row__line1 {
  font-size: 14px;
  line-height: 22px;
  margin-bottom: 6px;
}

.msg-reply-row__name {
  font-weight: 700;
  color: $c-text;
}

.msg-reply-row__name--tap {
  display: inline;
  margin: 0;
  padding: 0;
  border: none;
  background: none;
  font: inherit;
  font-weight: 700;
  color: inherit;
  cursor: pointer;
  text-align: left;
  vertical-align: baseline;
  &:hover {
    color: $c-blue;
  }
  &:focus-visible {
    outline: 2px solid rgba(0, 174, 236, 0.55);
    outline-offset: 1px;
    border-radius: 2px;
  }
}

.msg-reply-row__suffix {
  margin-left: 6px;
  color: $c-sub;
  font-weight: 400;
  font-size: 13px;
}

.msg-reply-row__content {
  margin: 0 0 10px;
  font-size: 14px;
  line-height: 22px;
  color: $c-text;
  word-break: break-word;
}

.msg-reply-row__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px 18px;
  font-size: 12px;
  line-height: 18px;
  color: $c-sub;
}

.msg-reply-row__time {
  color: $c-sub;
}

.msg-reply-row__act {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: none;
  background: none;
  color: $c-sub;
  font-size: 12px;
  cursor: pointer;
  &:hover:not(.msg-reply-row__act--liked) {
    color: $c-blue;
  }
}

.msg-reply-row__act--liked {
  color: $c-blue;
  font-weight: 600;
  &:hover {
    color: #0099cc;
  }
}

.msg-reply-row__ico {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  opacity: 0.92;
}

.msg-reply-row__ico--like-solid {
  opacity: 1;
}

.msg-reply-row__act--del {
  opacity: 1;
}

.msg-reply-compose {
  display: flex;
  align-items: stretch;
  gap: 12px;
  width: 100%;
  margin-top: 14px;
  box-sizing: border-box;
  min-width: 0;
  cursor: default;
}

.msg-reply-compose__lead {
  flex-shrink: 0;
  width: 48px;
  display: flex;
  justify-content: center;
}

.msg-reply-compose__face {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  background: #f1f2f3;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
}

.msg-reply-compose__field-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
}

.msg-reply-compose__textarea {
  flex: 1;
  width: 100%;
  min-width: 0;
  min-height: 64px;
  box-sizing: border-box;
  padding: 10px 12px;
  border: 1px solid rgba(0, 174, 236, 0.45);
  border-radius: 4px;
  background: #fff;
  cursor: text;
  font-size: 13px;
  line-height: 20px;
  color: $c-text;
  resize: vertical;
  font-family: inherit;
  &:focus {
    outline: none;
    border-color: $c-blue;
    box-shadow: 0 0 0 1px rgba(0, 174, 236, 0.25);
  }
  &::placeholder {
    color: $c-sub;
  }
}

.msg-reply-compose__submit {
  flex-shrink: 0;
  min-width: 92px;
  padding: 0 16px;
  border: none;
  border-radius: 4px;
  background: $c-blue;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  align-self: stretch;
  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
  &:hover:not(:disabled) {
    filter: brightness(1.05);
  }
}

.msg-reply-row__act--del:hover {
  color: #f25d8e;
}

.msg-reply-row__preview {
  flex-shrink: 0;
  width: 200px;
  max-width: 28%;
  font-size: 13px;
  line-height: 20px;
  color: $c-sub;
  text-align: right;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  word-break: break-word;
}

.msg-reply-row__preview--media {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  -webkit-line-clamp: unset;
  line-clamp: unset;
  -webkit-box-orient: unset;
}

.msg-reply-row__thumb-hit {
  display: block;
  padding: 0;
  margin: 0;
  border: none;
  background: transparent;
  cursor: pointer;
  text-align: right;
  font: inherit;
  color: inherit;
}

.msg-reply-row__thumb-wrap {
  flex-shrink: 0;
  display: block;
  width: 72px;
  height: 72px;
  border-radius: 2px;
  overflow: hidden;
  border: 1px solid #e5e6eb;
  background: #f1f2f3;
  box-sizing: border-box;
}

.msg-reply-row__thumb-wrap--empty {
  background: #f1f2f3;
}

.msg-reply-row__thumb {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.msg-reply-more {
  padding: 16px;
  text-align: center;
  border-top: 1px solid $c-line;
}

.msg-reply-more__btn {
  min-width: 120px;
  height: 34px;
  padding: 0 16px;
  border: 1px solid $c-line;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  color: $c-text;
  cursor: pointer;
  &:hover {
    border-color: $c-blue;
    color: $c-blue;
  }
}

/* —— 收到的赞列表（与主站「收到的赞」版式一致） —— */
.msg-like-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.msg-like-row {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 18px 20px;
  border-bottom: 1px solid #f0f0f0;
  box-sizing: border-box;
  cursor: pointer;
  &:last-child {
    border-bottom: none;
  }
}

.msg-like-row--unread {
  background: #fafbfd;
}

.msg-like-row__cols {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-width: 0;
}

.msg-like-row__faces {
  position: relative;
  flex-shrink: 0;
  width: 52px;
  height: 40px;
}

.msg-like-row__faces--single {
  width: 40px;
}

.msg-like-row__face {
  position: absolute;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  background: #f1f2f3;
  border: 2px solid #fff;
  box-sizing: border-box;
}

.msg-like-row__face--first {
  left: 0;
  top: 2px;
  z-index: 1;
}

.msg-like-row__face--second {
  left: 16px;
  top: 10px;
  z-index: 2;
}

.msg-like-row__face-link {
  display: block;
  text-decoration: none;
  color: inherit;
  overflow: hidden;
}

.msg-like-row__face-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  border-radius: 50%;
}

.msg-like-row__faces--single .msg-like-row__face--first {
  left: 2px;
  top: 2px;
}

.msg-like-row__mid {
  flex: 1;
  min-width: 0;
}

.msg-like-row__line1 {
  margin: 0;
  font-size: 14px;
  line-height: 22px;
  color: #333;
  word-break: break-word;
}

.msg-like-row__nick {
  font-weight: 600;
  color: #333;
}

.msg-like-row__nick-link {
  text-decoration: none;
  color: inherit;
  cursor: pointer;
  &:hover {
    color: #00a1d6;
  }
}

.msg-like-row__plain {
  font-weight: 400;
  color: #333;
}

.msg-like-row__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 14px;
  margin-top: 6px;
  font-size: 12px;
  line-height: 18px;
  color: #999;
}

.msg-like-row__time {
  color: #999;
}

.msg-like-row__acts {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px 16px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.12s ease;
}

.msg-like-row:hover .msg-like-row__acts {
  opacity: 1;
  pointer-events: auto;
}

.msg-like-row__act {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  margin: 0;
  border: none;
  background: none;
  font: inherit;
  font-size: 12px;
  line-height: 18px;
  color: #999;
  cursor: pointer;
  &:hover {
    color: #666;
  }
}

.msg-like-row__act-ico {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.msg-like-row__preview {
  flex-shrink: 0;
  width: 200px;
  max-width: 28%;
  font-size: 13px;
  line-height: 20px;
  color: #999;
  text-align: right;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  line-clamp: 4;
  -webkit-box-orient: vertical;
  word-break: break-word;
  align-self: center;
}

.msg-like-row__preview--tap {
  cursor: pointer;
  text-align: right;
  margin: 0;
  border: none;
  outline: none;
  box-shadow: none;
  background: transparent;
  appearance: none;
  -webkit-appearance: none;
  color: inherit;
  font: inherit;
  &:hover {
    color: $c-blue;
  }
  &:focus {
    outline: none;
  }
  &:focus-visible {
    outline: 2px solid rgba(0, 174, 236, 0.45);
    outline-offset: 2px;
  }
}

.msg-like-row__preview--media {
  width: 88px;
  max-width: 88px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: visible;
  -webkit-line-clamp: unset;
  line-clamp: unset;
}

.msg-like-row__thumb-wrap {
  display: block;
  width: 80px;
  height: 80px;
  border-radius: 6px;
  overflow: hidden;
  background: #f1f2f3;
  flex-shrink: 0;
}

.msg-like-row__thumb-wrap--empty {
  background: linear-gradient(135deg, #e8f7fd 0%, #f6f7f8 100%);
}

.msg-like-row__thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

/* —— 收到的赞 · 点赞详情：三块等宽白卡并列（灰底仅作间隙，不「套住」子卡） —— */
.msg-like-detail-stack {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow-x: hidden;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
  padding: 16px 20px 24px;
  box-sizing: border-box;
  background: #f6f7f8;
}

.msg-like-detail__panel {
  width: 100%;
  max-width: 100%;
  flex-shrink: 0;
  margin: 0;
  background: #fff;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  box-sizing: border-box;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

/* 点赞详情面包屑：与列表页「收到的赞」标题栏（msg-main__title-card + bar）同款 */
.msg-like-detail__crumb-nav {
  width: 100%;
  max-width: 100%;
  flex-shrink: 0;
}

.msg-like-detail__crumb-nav .msg-like-detail__crumb-link {
  margin: 0;
  padding: 0;
  border: none;
  background: none;
  cursor: pointer;
  font-family: inherit;
  font-size: 16px;
  font-weight: 500;
  line-height: 24px;
  color: #61666d;
  &:hover {
    color: $c-blue;
  }
}

.msg-like-detail__crumb-nav .msg-like-detail__crumb-sep {
  font-size: 16px;
  line-height: 24px;
  color: #c0c4cc;
  user-select: none;
}

.msg-like-detail__crumb-nav .msg-like-detail__crumb-here {
  font-size: 16px;
  font-weight: 500;
  line-height: 24px;
  color: #61666d;
}

.msg-like-detail__panel--ctx {
  padding: 0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}

.msg-like-detail__ctx-hit {
  display: block;
  width: 100%;
  margin: 0;
  padding: 14px 16px;
  border: none;
  background: none;
  text-align: left;
  font-size: 14px;
  line-height: 22px;
  font-weight: 400;
  color: #18191c;
  cursor: pointer;
  box-sizing: border-box;
  border-radius: 4px;
  font-family: inherit;
  &:hover {
    color: $c-blue;
    background: #fafbfd;
  }
}

.msg-like-detail__ctx-label {
  font-weight: 400;
  color: inherit;
}

.msg-like-detail__ctx-body {
  font-weight: 400;
  word-break: break-word;
  color: inherit;
}

.msg-like-detail__panel--list {
  padding: 0;
  overflow: hidden;
}

.msg-like-detail__loading,
.msg-like-detail__empty {
  margin: 0;
  padding: 28px 18px;
  text-align: center;
  font-size: 13px;
  color: $c-sub;
}

.msg-like-detail__ul {
  list-style: none;
  margin: 0;
  padding: 0;
}

.msg-like-detail__liker {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 16px;
  border-bottom: 1px solid #f0f0f0;
  box-sizing: border-box;
  &:last-child {
    border-bottom: none;
  }
}

.msg-like-detail__liker-face {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  overflow: hidden;
  background: #f1f2f3;
  text-decoration: none;
  display: block;
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
}

.msg-like-detail__liker-mid {
  flex: 1;
  min-width: 0;
}

.msg-like-detail__liker-line1 {
  font-size: 14px;
  line-height: 22px;
  color: #18191c;
}

.msg-like-detail__liker-name {
  font-weight: 600;
  color: #18191c;
  text-decoration: none;
  &:hover {
    color: $c-blue;
  }
}

.msg-like-detail__liker-suffix {
  margin-left: 4px;
  font-weight: 400;
  color: #18191c;
}

.msg-like-detail__liker-time {
  display: block;
  margin-top: 4px;
  font-size: 12px;
  line-height: 18px;
  color: #9499a0;
}

.msg-like-detail__follow {
  flex-shrink: 0;
  height: 28px;
  padding: 0 14px;
  border: 1px solid #e3e5e7;
  border-radius: 4px;
  background: #fff;
  font-size: 13px;
  color: #18191c;
  cursor: pointer;
  &:hover:not(:disabled) {
    border-color: #c9ccd0;
    background: #fafbfd;
  }
  &:disabled {
    cursor: default;
    opacity: 0.65;
  }
  &.is-followed {
    color: #9499a0;
    background: #f6f7f8;
    &:hover:not(:disabled) {
      color: #f25d8e;
      border-color: #f25d8e;
      background: #fff;
    }
  }
}

.msg-like-detail__more {
  padding: 12px 16px 16px;
  text-align: center;
  border-top: 1px solid #f0f0f0;
}

/* —— 其它分类 —— */
.msg-generic-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 16px 16px;
  display: flex;
  flex-direction: column;
  background: #fff;
}

.msg-generic-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.msg-generic-row {
  padding: 16px 12px;
  border-bottom: 1px solid $c-line;
}

.msg-generic-row__msg {
  margin: 0 0 6px;
  font-size: 14px;
  color: $c-text;
  line-height: 22px;
}

.msg-generic-row__sub {
  margin: 0 0 8px;
  font-size: 13px;
  color: $c-sub;
}

.msg-generic-row__time {
  font-size: 12px;
  color: $c-sub;
}

/* —— 删除通知确认弹窗（回复我的） —— */
.msg-del-overlay {
  position: fixed;
  inset: 0;
  z-index: 9000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  box-sizing: border-box;
}

.msg-del-overlay__backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
}

.msg-del-modal {
  position: relative;
  z-index: 1;
  width: min(320px, calc(100vw - 48px));
  box-sizing: border-box;
  padding: 28px 22px 22px;
  border-radius: 12px;
  background: #fff;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06);
  text-align: center;
}

.msg-del-modal__close {
  position: absolute;
  top: 10px;
  right: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: #9499a0;
  cursor: pointer;
  &:hover:not(:disabled) {
    color: #61666d;
    background: #f1f2f3;
  }
  &:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }
}

.msg-del-modal__title {
  margin: 4px 28px 14px;
  font-size: 17px;
  font-weight: 700;
  line-height: 24px;
  color: #18191c;
}

.msg-del-modal__desc {
  margin: 0 4px 22px;
  font-size: 14px;
  line-height: 22px;
  color: #666;
}

.msg-del-modal__actions {
  display: flex;
  gap: 12px;
  justify-content: stretch;
}

.msg-del-modal__btn {
  flex: 1;
  min-height: 42px;
  margin: 0;
  padding: 0 12px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  box-sizing: border-box;
  transition: opacity 0.15s ease, background 0.15s ease, border-color 0.15s ease;
  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
}

.msg-del-modal__btn--ghost {
  border: 1px solid #ddd;
  background: #fff;
  color: #18191c;
  &:hover:not(:disabled) {
    border-color: #c9ccd0;
    background: #fafbfd;
  }
}

.msg-del-modal__btn--primary {
  border: none;
  background: $c-blue;
  color: #fff;
  &:hover:not(:disabled) {
    filter: brightness(1.03);
  }
}
</style>

<!-- 私信分栏配色：非 scoped，穿透子组件，确保顶栏/输入区灰底 -->
<style lang="scss">
.msg-main__body-card > .mb-dm-chat {
  flex: 1 1 auto;
  min-height: 0;
}

.msg-main__body-card .mb-dm-chat .msg-col-msg,
.msg-main__body-card .mb-dm-chat .msg-col-msg__hint,
.msg-main__body-card .mb-dm-chat .msg-thread-list {
  background-color: #fff !important;
}

.msg-main__body-card .mb-dm-chat .msg-col-detail--chat {
  background-color: #f8f9fa !important;
}

.msg-main__body-card .mb-dm-chat .msg-chat-head {
  background-color: #f8f9fa !important;
}

.msg-main__body-card .mb-dm-chat .msg-chat-compose {
  background-color: #f8f9fa !important;
}

.msg-main__body-card .mb-dm-chat .msg-chat-scroll {
  background-color: #f8f9fa !important;
}

.msg-main__body-card .mb-dm-chat .msg-chat-compose-box {
  background-color: #fff !important;
}

.msg-main__body-card .mb-dm-chat .msg-chat-input {
  background-color: transparent !important;
}
</style>
