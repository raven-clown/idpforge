"use client";

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Users as UsersIcon } from "lucide-react";
import { Badge, Button, Flash, H1, H2, Input, Label, Panel, SelectOrCustom, TableWrap } from "@/components/ui";

const CREDENTIAL_TYPES = ["card", "face_2d", "face_3d", "fingerprint", "iris"];
import { api, ApiError, DeviceCredential, Role, User } from "@/lib/api";
import { formatDateTime } from "@/lib/time";
import { useCan } from "@/lib/current-user";

function UserDetail({ id }: { id: string }) {
  const canManageUsers = useCan("users", "manage");
  const canManageRbac = useCan("rbac", "manage");
  const [user, setUser] = useState<User | null>(null);
  const [userRoles, setUserRoles] = useState<Role[]>([]);
  const [allRoles, setAllRoles] = useState<Role[]>([]);
  const [credentials, setCredentials] = useState<DeviceCredential[]>([]);
  const [selectedRole, setSelectedRole] = useState("");
  const [credType, setCredType] = useState("");
  const [credRef, setCredRef] = useState("");
  const [credLabel, setCredLabel] = useState("");
  const [employeeIdDraft, setEmployeeIdDraft] = useState("");
  const [flash, setFlash] = useState<{ msg: string; kind: "ok" | "error" } | null>(null);

  function refresh() {
    api.users.get(id).then((u) => {
      setUser(u);
      setEmployeeIdDraft(u.employee_id ?? "");
    });
    api.rbac.userRoles(id).then((r) => setUserRoles(r.roles ?? []));
    api.rbac.roles().then((r) => setAllRoles(r.roles ?? []));
    api.users.credentials(id).then((r) => setCredentials(r.credentials ?? []));
  }

  useEffect(refresh, [id]);

  function report(action: () => Promise<unknown>, okMsg: string, failMsg: string) {
    action()
      .then(() => {
        setFlash({ msg: okMsg, kind: "ok" });
        refresh();
      })
      .catch((err) => setFlash({ msg: err instanceof ApiError ? err.message : failMsg, kind: "error" }));
  }

  if (!user) return <p className="text-muted text-sm">Loading...</p>;

  return (
    <>
      <div className="flex items-center gap-4 mb-6">
        {user.avatar_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={user.avatar_url} alt="" className="w-16 h-16 rounded-full object-cover border-2 border-border" />
        ) : null}
        <H1 icon={<UsersIcon size={20} />}>{user.username}</H1>
      </div>
      {flash && <Flash message={flash.msg} kind={flash.kind} />}

      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <tbody>
              <tr className="border-b border-border">
                <td className="py-2 px-3 text-muted">Email</td>
                <td className="py-2 px-3">{user.email}</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-2 px-3 text-muted">Employee ID</td>
                <td className="py-2 px-3">
                  {canManageUsers ? (
                    <div className="flex items-center gap-2">
                      <Input
                        value={employeeIdDraft}
                        onChange={(e) => setEmployeeIdDraft(e.target.value)}
                        placeholder="e.g. EMP-00123"
                        className="max-w-[200px]"
                      />
                      {employeeIdDraft !== (user.employee_id ?? "") && (
                        <Button
                          variant="secondary"
                          onClick={() =>
                            report(
                              () => api.users.update(id, { employee_id: employeeIdDraft }),
                              "Employee ID updated",
                              "Could not update employee ID"
                            )
                          }
                        >
                          Save
                        </Button>
                      )}
                    </div>
                  ) : (
                    user.employee_id || <span className="text-muted">-</span>
                  )}
                </td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-2 px-3 text-muted">Status</td>
                <td className="py-2 px-3">
                  <Badge>{user.status}</Badge>
                </td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-2 px-3 text-muted">MFA enabled</td>
                <td className="py-2 px-3">{user.mfa_enabled ? "yes" : "no"}</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-2 px-3 text-muted">Source</td>
                <td className="py-2 px-3">{user.source}</td>
              </tr>
              <tr className="border-b border-border">
                <td className="py-2 px-3 text-muted">Password</td>
                <td className="py-2 px-3">
                  {user.force_password_change ? (
                    <Badge tone="danger">must change on next login</Badge>
                  ) : (
                    <Badge tone="ok">up to date</Badge>
                  )}
                </td>
              </tr>
              <tr>
                <td className="py-2 px-3 text-muted">Created</td>
                <td className="py-2 px-3">{formatDateTime(user.created_at)}</td>
              </tr>
            </tbody>
          </table>
        </TableWrap>
        {canManageUsers && (
          <div className="flex gap-2.5 mt-4">
            <Button
              variant="secondary"
              onClick={() =>
                confirm("Reset this user's password to the organization default? They will be required to change it on next login.") &&
                report(
                  () => api.users.resetPassword(id),
                  "Password reset to the default; user must change it on next login",
                  "Could not reset password"
                )
              }
            >
              Reset to default password
            </Button>
            <Button
              variant="danger"
              onClick={() =>
                confirm("Disable this account?") &&
                report(() => api.users.offboard(id), "User offboarded", "Could not offboard user")
              }
            >
              Offboard
            </Button>
            <Button
              variant="danger"
              onClick={() =>
                confirm("Permanently delete this user?") &&
                report(() => api.users.remove(id), "User deleted", "Could not delete user")
              }
            >
              Delete
            </Button>
          </div>
        )}
      </Panel>

      <H2>Assigned roles</H2>
      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <thead>
              <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
                <th className="py-2 px-3">Role</th>
                <th className="py-2 px-3"></th>
              </tr>
            </thead>
            <tbody>
              {userRoles.map((r) => (
                <tr key={r.id} className="border-t border-border">
                  <td className="py-2 px-3">{r.name}</td>
                  <td className="py-2 px-3">
                    {canManageRbac && (
                      <Button
                        variant="secondary"
                        onClick={() =>
                          report(() => api.rbac.removeRoleFromUser(id, r.id), "Role removed", "Could not remove role")
                        }
                      >
                        Remove
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableWrap>
        {canManageRbac && (
          <div className="flex gap-2.5 items-end mt-3">
            <div className="flex-1">
              <Label>Assign role</Label>
              <select
                value={selectedRole}
                onChange={(e) => setSelectedRole(e.target.value)}
                className="w-full px-3 py-2 bg-input-bg border border-border rounded-lg text-sm text-text"
              >
                <option value="">Select a role</option>
                {allRoles.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name}
                  </option>
                ))}
              </select>
            </div>
            <Button
              onClick={() =>
                selectedRole &&
                report(() => api.rbac.assignRoleToUser(id, selectedRole), "Role assigned", "Could not assign role")
              }
            >
              Assign
            </Button>
          </div>
        )}
      </Panel>

      <H2>Device credentials (card / face / fingerprint)</H2>
      <Panel>
        <TableWrap>
          <table className="w-full border-collapse">
            <thead>
              <tr className="text-left text-[11.5px] uppercase tracking-wide text-muted">
                <th className="py-2 px-3">Type</th>
                <th className="py-2 px-3">Reference</th>
                <th className="py-2 px-3">Label</th>
                <th className="py-2 px-3"></th>
              </tr>
            </thead>
            <tbody>
              {credentials.map((c) => (
                <tr key={c.id} className="border-t border-border">
                  <td className="py-2 px-3">{c.credential_type}</td>
                  <td className="py-2 px-3 font-mono text-xs">{c.credential_ref}</td>
                  <td className="py-2 px-3">{c.label}</td>
                  <td className="py-2 px-3">
                    {canManageUsers && (
                      <Button
                        variant="secondary"
                        onClick={() =>
                          report(
                            () => api.users.removeCredential(id, c.id),
                            "Credential removed",
                            "Could not remove credential"
                          )
                        }
                      >
                        Remove
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableWrap>
        {canManageUsers && (
          <>
            <Label>Type</Label>
            <SelectOrCustom value={credType} onChange={setCredType} options={CREDENTIAL_TYPES} placeholder="Custom credential type" />
            <Label>Reference (card number / device template ID, never raw biometric data)</Label>
            <Input value={credRef} onChange={(e) => setCredRef(e.target.value)} />
            <Label>Label</Label>
            <Input value={credLabel} onChange={(e) => setCredLabel(e.target.value)} />
            <Button
              className="mt-4"
              onClick={() => {
                if (!credType || !credRef) return;
                report(
                  () => api.users.addCredential(id, credType, credRef, credLabel),
                  "Credential added",
                  "Could not add credential (already enrolled to someone?)"
                );
                setCredType("");
                setCredRef("");
                setCredLabel("");
              }}
            >
              Add
            </Button>
          </>
        )}
      </Panel>
    </>
  );
}

function UserDetailInner() {
  const params = useSearchParams();
  const id = params.get("id") ?? "";
  if (!id) return <p className="text-muted text-sm">No user selected.</p>;
  return <UserDetail id={id} />;
}

export default function UserDetailPage() {
  return (
    <Suspense fallback={<p className="text-muted text-sm">Loading...</p>}>
      <UserDetailInner />
    </Suspense>
  );
}
