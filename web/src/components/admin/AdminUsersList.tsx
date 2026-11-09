import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../../stores/auth";
import { listAdminUsers, setUserVerified, updateUserRole, type AdminUser } from "../../lib/api/admin";
import type { UserRole } from "../../lib/api/auth";
import { ApiError } from "../../lib/api/client";
import IslandErrorBoundary from "../IslandErrorBoundary";
import AdminUserDetailModal from "./AdminUserDetailModal";

const roleLabels: Record<UserRole, string> = {
    ADMIN: "Admin",
    COMMUNITY_MEMBER: "Miembro de comunidad",
    PAID_MEMBER: "Miembro pago",
    PUBLIC_MEMBER: "Público",
};

const roleOptions = Object.entries(roleLabels) as [UserRole, string][];

export default function AdminUsersList() {
    return (
        <IslandErrorBoundary>
            <AdminUsersListContent />
        </IslandErrorBoundary>
    );
}

function AdminUsersListContent() {
    const auth = useStore($auth);
    const [users, setUsers] = useState<AdminUser[] | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [updatingId, setUpdatingId] = useState<number | null>(null);
    const [updateError, setUpdateError] = useState<string | null>(null);
    // Se guarda el id y no el objeto: así, tras un refetch, el modal
    // muestra la versión nueva del usuario sin sincronizar nada a mano.
    const [detailUserId, setDetailUserId] = useState<number | null>(null);

    useEffect(() => {
        if (auth.status === "loading") void initAuth();
    }, [auth.status]);

    useEffect(() => {
        if (auth.status !== "authenticated" || auth.user.role !== "ADMIN") return;
        listAdminUsers()
            .then(setUsers)
            .catch((err) => setError(err instanceof ApiError ? err.message : "No pudimos cargar los usuarios."));
    }, [auth.status]);

    // Sin edición optimista: igual que AdminCourseEditor, cada cambio de rol
    // vuelve a pedir la lista entera en vez de mutar el array local a mano.
    // Ojo: la recarga posterior al cambio usa `updateError`, no `error` — si
    // el PATCH funcionó pero el refetch falla, no queremos que la tabla
    // entera desaparezca detrás de un mensaje de error (`error` esconde
    // toda la tabla, así que se reserva para la carga inicial).
    async function handleRoleChange(userId: number, role: UserRole) {
        setUpdateError(null);
        setUpdatingId(userId);
        try {
            await updateUserRole(userId, role);
            const data = await listAdminUsers();
            setUsers(data);
        } catch (err) {
            setUpdateError(err instanceof ApiError ? err.message : "No se pudo actualizar el rol.");
        } finally {
            setUpdatingId(null);
        }
    }

    /** Mismo criterio que el cambio de rol: PATCH y después refetch. El
     * error se propaga al modal, que lo muestra junto al toggle. */
    async function handleVerifiedChange(userId: number, verified: boolean) {
        await setUserVerified(userId, verified);
        setUsers(await listAdminUsers());
    }

    if (auth.status === "loading") {
        return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
    }

    if (auth.status === "anonymous") {
        return (
            <div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
                <p className="font-serif text-ink-80">Iniciá sesión con una cuenta de administrador para ver esto.</p>
                <a href="/ingresar" className="btn btn-cta">
                    Ingresar
                </a>
            </div>
        );
    }

    if (auth.user.role !== "ADMIN") {
        return (
            <p role="alert" className="font-ui text-sm text-sacred-red">
                Esta sección es solo para administradores.
            </p>
        );
    }

    const detailUser = detailUserId !== null ? users?.find((u) => u.id === detailUserId) : undefined;

    return (
        <div className="flex flex-col gap-6">
            {error ? (
                <p role="alert" className="font-ui text-sm text-sacred-red">
                    {error}
                </p>
            ) : !users ? (
                <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando usuarios" />
            ) : users.length === 0 ? (
                <p className="font-ui text-sm text-ink-60">Todavía no hay usuarios cargados.</p>
            ) : (
                <>
                    {updateError && (
                        <p role="alert" className="font-ui text-sm text-sacred-red">
                            {updateError}
                        </p>
                    )}
                    <div className="overflow-x-auto rounded-[4px] border border-ink-15">
                        <table className="table table-sm font-ui">
                            <thead>
                                <tr className="text-ink-60">
                                    <th>Id</th>
                                    <th>Email</th>
                                    <th>Nombre</th>
                                    <th>Servicio Auth</th>
                                    <th>Verificado</th>
                                    <th>Rol</th>
                                    <th>Fecha Alta</th>
                                    <th>
                                        <span className="sr-only">Acciones</span>
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {users.map((user) => {
                                    const isSelf = user.id === auth.user.id;
                                    const roleKnown = roleOptions.some(([value]) => value === user.role);
                                    return (
                                        <tr key={user.id}>
                                            <td className="text-ink-60">{user.id}</td>
                                            <td>{user.email}</td>
                                            <td>{user.name}</td>
                                            <td>{user.authProvider}</td>
                                            <td>
                                                {user.emailVerified ? (
                                                    <span className="badge badge-success badge-sm">Verificado</span>
                                                ) : (
                                                    <span className="badge badge-ghost badge-sm">Sin Verificar</span>
                                                )}
                                            </td>
                                            <td>
                                                <select
                                                    value={user.role}
                                                    disabled={isSelf || updatingId === user.id}
                                                    title={isSelf ? "No podés cambiar tu propio rol." : undefined}
                                                    onChange={(e) => void handleRoleChange(user.id, e.target.value as UserRole)}
                                                    className="select select-bordered select-sm"
                                                >
                                                    {/* Si el rol del usuario no está en roleLabels (ej. un rol nuevo
                                                        agregado en el backend pero no acá todavía), lo agregamos como
                                                        opción cruda — sin esto, el <select> no encontraría ningún
                                                        <option> con ese value y el navegador mostraría la primera
                                                        opción de la lista como si estuviera seleccionada, dando a
                                                        entender un rol distinto al real. */}
                                                    {!roleKnown && <option value={user.role}>{user.role} (rol desconocido)</option>}
                                                    {roleOptions.map(([value, label]) => (
                                                        <option key={value} value={value}>
                                                            {label}
                                                        </option>
                                                    ))}
                                                </select>
                                            </td>
                                            <td className="text-ink-60">{formatDate(user.createdAt)}</td>
                                            <td>
                                                <button
                                                    type="button"
                                                    onClick={() => setDetailUserId(user.id)}
                                                    className="btn btn-outline btn-xs"
                                                >
                                                    Detalles
                                                </button>
                                            </td>
                                        </tr>
                                    );
                                })}
                            </tbody>
                        </table>
                    </div>
                </>
            )}

            {detailUser && (
                <AdminUserDetailModal
                    key={detailUser.id}
                    user={detailUser}
                    roleLabel={roleLabels[detailUser.role] ?? detailUser.role}
                    onVerifiedChange={(verified) => handleVerifiedChange(detailUser.id, verified)}
                    onClose={() => setDetailUserId(null)}
                />
            )}
        </div>
    );
}

function formatDate(value: string): string {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? "—" : date.toLocaleDateString("es-AR");
}
