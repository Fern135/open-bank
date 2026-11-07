import type { Metadata } from "next";
import { REPO_URL } from "../../components/legalLinks";

export const metadata: Metadata = { title: "Terms of service" };

export default function Terms() {
  return (
    <>
      <h1>Terms of service</h1>
      <p>Last updated November 7, 2026</p>
      <p>
        These terms apply when you use this website. Your deposit account
        agreement applies to any account you hold with us.
      </p>
      <h2>Using this site</h2>
      <p>
        You may use this site for lawful, personal purposes. You may not try to
        disrupt it, gain access to data that isn&apos;t yours, or use it to
        mislead anyone.
      </p>
      <h2>No financial advice</h2>
      <p>
        Information on this site is general. It isn&apos;t financial, legal or
        tax advice.
      </p>
      <h2>Availability</h2>
      <p>We may change, suspend or stop any part of this site at any time.</p>
      <h2>No warranty</h2>
      <p>
        This site is provided as is, without warranties of any kind, to the
        extent the law allows. We aren&apos;t liable for losses that come from
        using it, to the extent the law allows.
      </p>
      <h2>Changes to these terms</h2>
      <p>
        We&apos;ll post any change here and update the date above. If you keep
        using the site after a change, the new terms apply.
      </p>
      <h2>Contact</h2>
      <p>
        Send questions through our{" "}
        <a href={`${REPO_URL}/issues`}>support page</a>.
      </p>
    </>
  );
}
