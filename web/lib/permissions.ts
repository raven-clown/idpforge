// Mirrors internal/rbac/models.go's Permission.matches: a bare "*"
// resource or a "resource/*" prefix grants everything under it, otherwise
// it's an exact match. Permission strings come from /api/v1/me as
// "resource:action".
export function hasPermission(permissions: string[], resource: string, action: string): boolean {
  return permissions.some((p) => {
    const sep = p.indexOf(":");
    if (sep < 0) return false;
    const res = p.slice(0, sep);
    const act = p.slice(sep + 1);
    if (act !== action) return false;
    if (res === resource || res === "*") return true;
    if (res.endsWith("/*")) {
      const prefix = res.slice(0, -2);
      return resource === prefix || resource.startsWith(prefix + "/");
    }
    return false;
  });
}
