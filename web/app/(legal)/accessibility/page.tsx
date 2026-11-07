import type { Metadata } from "next";
import { REPO_URL } from "../../components/legalLinks";

export const metadata: Metadata = { title: "Accessibility" };

export default function Accessibility() {
  return (
    <>
      <h1>Accessibility</h1>
      <p>Last updated November 7, 2026</p>
      <p>
        Everyone should be able to use their bank. We aim to meet the Web
        Content Accessibility Guidelines (WCAG) 2.2 at level AA.
      </p>
      <h2>What this site supports</h2>
      <ul>
        <li>Keyboard navigation, with a visible focus outline</li>
        <li>
          Screen reader announcements when a ledger transaction posts or is
          declined
        </li>
        <li>Reduced motion, if you turn it on in your device settings</li>
        <li>Text that can be zoomed without losing content</li>
      </ul>
      <h2>Tell us about a barrier</h2>
      <p>
        If something is hard to use, tell us through our{" "}
        <a href={`${REPO_URL}/issues`}>support page</a>. Include the page and
        the device or assistive technology you were using.
      </p>
    </>
  );
}
