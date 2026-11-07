import SiteFooter from "../components/SiteFooter";
import SiteHeader from "../components/SiteHeader";

export default function SiteLayout({ children }: LayoutProps<"/">) {
  return (
    <>
      <SiteHeader />
      <main id="main">{children}</main>
      <SiteFooter />
    </>
  );
}
