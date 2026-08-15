import { useEffect } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth, logout } from "../stores/auth";

/** The only interactive piece of the navbar: resolves the session on mount
 * and swaps the CTA between "sign up" and "signed in" states. Everything
 * else in the navbar is static markup rendered by Navbar.astro. */
export default function AuthStatus() {
	const auth = useStore($auth);

	useEffect(() => {
		if (auth.status === "loading") {
			void initAuth();
		}
	}, [auth.status]);

	if (auth.status === "loading") {
		return <span className="loading loading-spinner loading-sm text-marian-blue" aria-label="Cargando sesión" />;
	}

	if (auth.status === "authenticated") {
		return (
			<div className="flex items-center gap-3">
				{auth.user.role === "ADMIN" && (
					<a href="/admin/cursos" className="font-ui text-sm text-marian-blue hover:text-marian-blue-deep">
						Admin
					</a>
				)}
				<span className="font-ui text-sm text-ink-80">{auth.user.name}</span>
				<button type="button" className="btn btn-outline btn-sm" onClick={() => void logout()}>
					Salir
				</button>
			</div>
		);
	}

	return (
		<div className="flex items-center gap-3">
			<a href="/ingresar" className="font-ui text-sm text-marian-blue hover:text-marian-blue-deep">
				Ingresar
			</a>
			<a href="/registro" className="btn btn-cta btn-sm">
				Registrarse
			</a>
		</div>
	);
}
