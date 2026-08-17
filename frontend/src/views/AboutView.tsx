import { ExternalLink, Network } from "lucide-react";
import { useI18n } from "../lib/i18n";

const upstreamProjects = [
  {
    id: "tailscale",
    name: "Tailscale",
    url: "https://tailscale.com/",
    displayUrl: "tailscale.com",
  },
  {
    id: "headscale",
    name: "Headscale",
    url: "https://headscale.net/",
    displayUrl: "headscale.net",
  },
] as const;

export function AboutView({ version, onOpenURL }: { version: string; onOpenURL: (url: string) => Promise<void> }) {
  const { t } = useI18n();

  return (
    <div className="view-stack about-view">
      <section className="about-product" aria-labelledby="about-product-title">
        <div className="about-product-heading">
          <span className="about-product-mark">
            <Network aria-hidden="true" size={24} strokeWidth={2} />
          </span>
          <div>
            <h2 id="about-product-title">HeadscaleClient</h2>
            <p>{t("about.versionValue", { version })}</p>
          </div>
        </div>
        <dl className="about-facts">
          <div><dt>{t("about.publisher")}</dt><dd>BIMCC., Ltd.</dd></div>
          <div><dt>{t("about.copyright")}</dt><dd>(c) 2026 BIMCC., Ltd.</dd></div>
        </dl>
        <p className="about-independence">{t("about.independence")}</p>
      </section>

      <section className="section-block" aria-labelledby="upstream-projects-title">
        <header className="section-header">
          <div>
            <h2 id="upstream-projects-title">{t("about.upstreamProjects")}</h2>
            <p>{t("about.upstreamHint")}</p>
          </div>
        </header>
        <div className="upstream-list">
          {upstreamProjects.map((project) => (
            <article className="upstream-row" key={project.id}>
              <div>
                <div className="upstream-title">
                  <h3>{project.name}</h3>
                  <span>BSD 3-Clause</span>
                </div>
                <p>{t(`about.${project.id}Copyright`)}</p>
              </div>
              <button
                className="button secondary with-icon upstream-link"
                type="button"
                aria-label={t("about.openWebsite", { name: project.name, url: project.displayUrl })}
                onClick={() => void onOpenURL(project.url).catch(() => undefined)}
              >
                {project.displayUrl}
                <ExternalLink aria-hidden="true" size={16} />
              </button>
            </article>
          ))}
        </div>
      </section>
    </div>
  );
}
