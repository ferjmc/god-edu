import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../../stores/auth";
import { listAdminUsers, updateUserRole, type AdminUser } from "../../lib/api/admin";
import type { UserRole } from "../../lib/api/auth";
import { ApiError } from "../../lib/api/client";
//import NewCourseForm from "./NewCourseForm";

const roleLabels: Record<UserRole, string> = {
    ADMIN: "Admin",
    COMMUNITY_MEMBER: "Miembro de comunidad",
    PAID_MEMBER: "Miembro pago",
    PUBLIC_MEMBER: "Público",
};

const roleOptions = Object.entries(roleLabels) as [UserRole, string][];

export default function AdminUsersList() {
    const auth = useStore($auth);
    const [users, setUsers] = useState<AdminUser[] | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [updatingId, setUpdatingId] = useState<number | null>(null);
    const [updateError, setUpdateError] = useState<string | null>(null);

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
    // entera desaparezca detrás de un mensaje de error (ver AdminUsersList.tsx
    // línea 86: `error` esconde toda la tabla, así que reservarlo para la
    // carga inicial).
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

    return (
        <div className="flex flex-col gap-6">
            {/* <NewCourseForm onCreated={(slug) => window.location.assign(`/admin/cursos/${slug}`)} /> */}

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
                    <table className="w-full border-collapse font-ui text-sm">
                        <thead>
                            <tr className="border-b border-ink-15 text-left text-ink-60">
                                <th className="py-2 pr-4 font-medium">Id</th>
                                <th className="py-2 pr-4 font-medium">Email</th>
                                <th className="py-2 pr-4 font-medium">Nombre</th>
                                <th className="py-2 pr-4 font-medium">Servicio Auth</th>
                                <th className="py-2 pr-4 font-medium">Verificado</th>
                                <th className="py-2 pr-4 font-medium">Rol</th>
                                <th className="py-2 pr-4 font-medium">Fecha Alta</th>
                            </tr>
                        </thead>
                        <tbody>
                            {users.map((user) => {
                                const isSelf = user.id === auth.user.id;
                                const roleKnown = roleOptions.some(([value]) => value === user.role);
                                return (
                                    <tr key={user.id} className="border-b border-ink-15/60">
                                        <td className="py-2.5 pr-4 text-ink-60">{user.id}</td>
                                        <td className="py-2.5 pr-4">{user.email}</td>
                                        <td className="py-2.5 pr-4">{user.name}</td>
                                        <td className="py-2.5 pr-4">{user.authProvider}</td>
                                        <td className="py-2.5 pr-4">
                                            {user.emailVerified ? (
                                                <span className="badge badge-success badge-sm">Verificado</span>
                                            ) : (
                                                <span className="badge badge-ghost badge-sm">Sin Verificar</span>
                                            )}
                                        </td>
                                        <td className="py-2.5 pr-4">
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
                                        <td className="py-2.5 pr-4 text-ink-60">{new Date(user.createdAt).toLocaleDateString("es-AR")}</td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </>
            )}
        </div>
    );
}
