import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../../stores/auth";
import { listAdminUsers, type AdminUser } from "../../lib/api/admin";
import { ApiError } from "../../lib/api/client";
//import NewCourseForm from "./NewCourseForm";

export default function AdminUsersList() {
    const auth = useStore($auth);
    const [users, setUsers] = useState<AdminUser[] | null>(null);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (auth.status === "loading") void initAuth();
    }, [auth.status]);

    useEffect(() => {
        if (auth.status !== "authenticated" || auth.user.role !== "ADMIN") return;
        listAdminUsers()
            .then(setUsers)
            .catch((err) => setError(err instanceof ApiError ? err.message : "No pudimos cargar los usuarios."));
    }, [auth.status]);

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
                        {users.map((user) => (
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
                                <td className="py-2.5 pr-4">{user.role}</td>
                                <td className="py-2.5 pr-4 text-ink-60">{new Date(user.createdAt).toLocaleDateString("es-AR")}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </div>
    );
}
