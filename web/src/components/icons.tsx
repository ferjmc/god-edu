/** Íconos UI mínimos, dibujados a mano — mismo criterio que los íconos de
 * redes sociales del footer: no se suma una librería (lucide, heroicons)
 * por un puñado de glifos funcionales. Outline fino (stroke 1.6), no
 * sólidos, para diferenciarlos de los brand marks del footer. */

type IconProps = { className?: string };

export function CheckCircleIcon({ className }: IconProps) {
	return (
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" className={className} aria-hidden="true">
			<circle cx="12" cy="12" r="9" />
			<path d="M8.5 12.5l2.5 2.5 4.5-5" strokeLinecap="round" strokeLinejoin="round" />
		</svg>
	);
}

export function DownloadIcon({ className }: IconProps) {
	return (
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" className={className} aria-hidden="true">
			<path d="M12 4v10m0 0l-3.5-3.5M12 14l3.5-3.5" strokeLinecap="round" strokeLinejoin="round" />
			<path d="M5 18h14" strokeLinecap="round" />
		</svg>
	);
}

export function FileIcon({ className }: IconProps) {
	return (
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" className={className} aria-hidden="true">
			<path d="M7 3.5h6.5L18 8v12.5H7z" strokeLinejoin="round" />
			<path d="M13.5 3.5V8H18" strokeLinejoin="round" />
		</svg>
	);
}

export function XCircleIcon({ className }: IconProps) {
	return (
		<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" className={className} aria-hidden="true">
			<circle cx="12" cy="12" r="9" />
			<path d="M9.5 9.5l5 5m0-5l-5 5" strokeLinecap="round" />
		</svg>
	);
}
