import { useEffect, useState } from "react";
import QRCode from "qrcode";
import { verifyCertificate, type Certificate } from "../lib/api/certificates";
import { formatLongDate } from "../lib/format";
import { CheckCircleIcon, XCircleIcon } from "./icons";

type Status = "missing" | "verifying" | "success" | "error";

/** Lee ?code= de la URL (window.location, no props — esta página no tiene
 * ruta dinámica en Astro: el sitio se despliega estático y un código de
 * certificado no existe todavía al momento del build, así que no puede ser
 * un [code].astro con getStaticPaths — mismo problema y misma solución que
 * VerifyEmailStatus con ?token=). Pública a propósito: cualquiera con el
 * link o el QR tiene que poder verificar un certificado sin tener cuenta. */
export default function CertificateView() {
	const [status, setStatus] = useState<Status>("verifying");
	const [certificate, setCertificate] = useState<Certificate | null>(null);
	const [qrDataUrl, setQrDataUrl] = useState<string | null>(null);

	useEffect(() => {
		const code = new URLSearchParams(window.location.search).get("code");
		if (!code) {
			setStatus("missing");
			return;
		}
		verifyCertificate(code)
			.then((cert) => {
				setCertificate(cert);
				setStatus("success");
			})
			.catch(() => setStatus("error"));
	}, []);

	useEffect(() => {
		if (status !== "success") return;
		// El QR codifica esta misma URL pública — escanearlo vuelve a esta
		// pantalla, que vuelve a verificar contra el API. Se genera en el
		// navegador con la librería `qrcode`: no hace falta ningún endpoint
		// nuevo ni ninguna dependencia en el backend para esto.
		QRCode.toDataURL(window.location.href, { margin: 1, width: 220 })
			.then(setQrDataUrl)
			.catch(() => setQrDataUrl(null));
	}, [status]);

	if (status === "verifying") {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Verificando" />;
	}

	if (status === "missing") {
		return (
			<div className="flex flex-col items-start gap-3 rounded-[4px] border border-ink-15 bg-paper p-8">
				<XCircleIcon className="h-10 w-10 text-sacred-red" />
				<p className="font-serif text-ink-80">
					Este link no trae ningún código de certificado. Revisá que hayas copiado la URL completa.
				</p>
			</div>
		);
	}

	if (status === "error") {
		return (
			<div className="flex flex-col items-start gap-4 rounded-[4px] border border-ink-15 bg-paper p-8">
				<XCircleIcon className="h-10 w-10 text-sacred-red" />
				<p className="font-serif text-ink-80">No encontramos ningún certificado con ese código.</p>
			</div>
		);
	}

	// status === "success", pero certificate todavía puede no estar seteado
	// en el primer render de este estado — evita un "!" a costa de un frame
	// extra de loading, no de un error de tipos.
	if (!certificate) {
		return <span className="loading loading-spinner loading-md text-marian-blue" aria-label="Cargando" />;
	}

	return (
		<div className="flex w-full flex-col items-start gap-6 rounded-[4px] border border-ink-15 bg-paper p-8 shadow-gold-edge">
			<CheckCircleIcon className="h-10 w-10 text-liturgical-gold" />
			<div>
				<p className="font-ui text-xs font-semibold uppercase tracking-[0.14em] text-marian-blue">Certificado válido</p>
				<h2 className="mt-2 font-display text-2xl text-marian-blue-deep">{certificate.recipientName}</h2>
				<p className="mt-1 font-serif text-ink-80">completó el curso</p>
				<p className="font-display text-xl text-ink">{certificate.courseTitle}</p>
				<p className="mt-3 font-ui text-sm text-ink-60">Emitido el {formatLongDate(certificate.issuedAt)}</p>
			</div>
			{qrDataUrl && <img src={qrDataUrl} alt="Código QR de verificación" width={220} height={220} />}
		</div>
	);
}
