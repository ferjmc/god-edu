/** Única fuente de las secciones del panel admin: la usan el menú lateral
 * (AdminMenu.astro) y las tarjetas de /admin. Sumar una sección nueva es
 * agregar una entrada acá. */
export type AdminNavItem = {
	href: string;
	label: string;
	description: string;
};

export const adminNavItems: AdminNavItem[] = [
	{ href: "/admin", label: "Inicio", description: "Resumen del panel de administración." },
	{ href: "/admin/usuarios", label: "Usuarios", description: "Roles, verificación de cuenta y lecciones vistas por cada usuario." },
	{ href: "/admin/cursos", label: "Cursos", description: "Crear y editar cursos, lecciones y su contenido." },
];

/** "/admin" solo está activo en su propia página; el resto también en sus
 * subpáginas (ej. /admin/cursos/mi-curso deja activo "Cursos"). */
export function isAdminNavItemActive(item: AdminNavItem, pathname: string): boolean {
	const path = pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
	if (item.href === "/admin") return path === "/admin";
	return path === item.href || path.startsWith(`${item.href}/`);
}
