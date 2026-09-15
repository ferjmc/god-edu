import { useEffect, useState } from "react";
import { useStore } from "@nanostores/react";
import { $auth, initAuth } from "../stores/auth";
import { listMyCourses, type CourseProgress } from "../lib/api/courses";
import { ApiError } from "../lib/api/client";
import CourseProgressCard from "./CourseProgressCard";

export default function MisCursos() {
	const auth = useStore($auth);
	const [courses, setCourses] = useState<CourseProgress[] | null>(null);
	const [error, setError] = useState<string | null>(null);
	const [showWelcome, setShowWelcome] = useState(false);

	// Register redirects here with ?welcome=1 right after creating the
	// account — strip it from the URL so a refresh doesn't keep showing it.
	useEffect(() => {
		const params = new URLSearchParams(window.location.search);
		if (params.get("welcome") !== "1") return;
		setShowWelcome(true);
		params.delete("welcome");
		const query = params.toString();
		window.history.replaceState({}, "", `${window.location.pathname}${query ? `?${query}` : ""}`);
	}, []);

	useEffect(() => {
		if (auth.status === "loading") void initAuth();
	}, [auth.status]);

	useEffect(() => {
		if (auth.status !== "authenticated") return;
		listMyCourses()
			.then(setCourses)
			.catch((err) => setError(err instanceof ApiError ? err.message : "No pudimos cargar tus cursos."));
	}, [auth.status]);

	const welcomeBanner = showWelcome ? (
		<div
			role="status"
			className="mb-6 rounded-[4px] border border-marian-blue/30 bg-marian-blue/5 px-4 py-3 font-ui text-sm text-ink-80"
		>
			¡Cuenta creada! Te enviamos un correo para verificar tu dirección de email.
		</div>
	) : null;

	if (auth.status === "loading") {
		return (
			<>
				{welcomeBanner}
				<span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />
			</>
		);
	}

	if (auth.status === "anonymous") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<p className="font-serif text-ink-80">Iniciá sesión para ver tus cursos.</p>
				<a href="/ingresar" className="btn btn-cta">
					Ingresar
				</a>
			</div>
		);
	}

	if (error) {
		return (
			<>
				{welcomeBanner}
				<p role="alert" className="font-ui text-sm text-sacred-red">
					{error}
				</p>
			</>
		);
	}

	if (!courses) {
		return (
			<>
				{welcomeBanner}
				<span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando cursos" />
			</>
		);
	}

	if (courses.length === 0) {
		return (
			<>
				{welcomeBanner}
				<p className="font-ui text-sm text-ink-60">Todavía no tenés cursos disponibles.</p>
			</>
		);
	}

	return (
		<>
			{welcomeBanner}
			<div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
				{courses.map((c) => (
					<CourseProgressCard key={c.id} course={c} />
				))}
			</div>
		</>
	);
}
