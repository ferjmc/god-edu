import { useState, type FormEvent } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../stores/auth";
import { updateBirthDate } from "../lib/api/auth";
import { ApiError } from "../lib/api/client";

/** Pantalla a la que redirige el callback de OAuth cuando la cuenta todavía
 * no tiene fecha de nacimiento guardada — alta social nueva, o cuenta OAuth
 * vieja de antes de este campo (ver oauth.go, Callback). Requiere sesión:
 * el callback ya deja la cookie puesta antes de redirigir para acá. */
export default function CompleteProfile() {
	const auth = useStore($auth);
	const [birthDate, setBirthDate] = useState("");
	const [error, setError] = useState<string | null>(null);
	const [submitting, setSubmitting] = useState(false);

	if (auth.status === "loading") {
		void initAuth();
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
	}

	if (auth.status === "anonymous") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<p className="font-serif text-ink-80">Iniciá sesión para completar tu perfil.</p>
				<a href="/ingresar" className="btn btn-cta">
					Ingresar
				</a>
			</div>
		);
	}

	async function handleSubmit(e: FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setError(null);
		setSubmitting(true);
		try {
			await updateBirthDate(birthDate);
			window.location.assign("/mis-cursos?welcome=1");
		} catch (err) {
			setError(err instanceof ApiError ? err.message : "No se pudo guardar. Intentá de nuevo.");
			setSubmitting(false);
		}
	}

	return (
		<form onSubmit={handleSubmit} className="flex w-full max-w-sm flex-col gap-4">
			<label className="flex flex-col gap-1">
				<span className="font-ui text-sm text-ink-80">Fecha de nacimiento</span>
				<input
					type="date"
					required
					value={birthDate}
					onChange={(e) => setBirthDate(e.target.value)}
					className="input input-bordered w-full"
				/>
			</label>

			{error && (
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			)}

			<button type="submit" disabled={submitting} className="btn btn-primary w-full">
				{submitting ? "Guardando…" : "Continuar"}
			</button>
		</form>
	);
}
