const K_ACCESS = "cakecake_admin_access_token";
const K_REFRESH = "cakecake_admin_refresh_token";

export function getAdminAccessToken() {
  return localStorage.getItem(K_ACCESS) || "";
}

export function getAdminRefreshToken() {
  return localStorage.getItem(K_REFRESH) || "";
}

export function setAdminTokens(access, refresh) {
  if (access) {
    localStorage.setItem(K_ACCESS, access);
  }
  if (refresh) {
    localStorage.setItem(K_REFRESH, refresh);
  }
}

export function clearAdminTokens() {
  localStorage.removeItem(K_ACCESS);
  localStorage.removeItem(K_REFRESH);
}

export function isAdminLoggedIn() {
  return !!getAdminAccessToken();
}
