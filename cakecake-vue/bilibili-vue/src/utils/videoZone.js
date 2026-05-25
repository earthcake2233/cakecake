/** @param {string} raw */
export function parseVideoZone(raw) {
  let z = String(raw || "").trim();
  if (!z) {
    return { zone: "", parent: "", child: "", category: "" };
  }
  z = z.replace(/→/g, "-").replace(/\s*-\s*/g, "-");
  const idx = z.indexOf("-");
  if (idx > 0) {
    const parent = z.slice(0, idx).trim();
    const child = z.slice(idx + 1).trim();
    return {
      zone: z,
      parent,
      child,
      category: child ? `${parent} > ${child}` : parent
    };
  }
  return { zone: z, parent: z, child: "", category: z };
}

/** @param {string} raw */
export function videoZoneCrumbs(raw) {
  const { parent, child } = parseVideoZone(raw);
  if (!parent) {
    return [];
  }
  const crumbs = [{ key: "home", label: "主页", to: { name: "home" } }];
  crumbs.push({ key: "parent", label: parent });
  if (child) {
    crumbs.push({ key: "child", label: child, last: true });
  } else {
    crumbs[crumbs.length - 1].last = true;
  }
  return crumbs;
}
