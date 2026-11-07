import LegalNav from "./LegalNav";
import styles from "./legal.module.css";

export default function LegalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="container pt-4">
      <div className="row gy-4">
        <div className="col-lg-3">
          <LegalNav />
        </div>
        <article className={`col-lg-8 offset-lg-1 ${styles.prose}`}>
          {children}
        </article>
      </div>
    </div>
  );
}
