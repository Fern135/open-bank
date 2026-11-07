import type { Metadata } from "next";
import { REPO_URL } from "../../../components/legalLinks";

export const metadata: Metadata = { title: "Privacy notice" };

export default function Privacy() {
  return (
    <>
      <h1>Privacy notice</h1>
      <p>Last updated November 7, 2026</p>
      <p>
        Federal law gives you the right to limit some sharing of your personal
        information. This notice explains what we would collect, how we would
        share it, and what you can limit.
      </p>
      <h2>What we collect today</h2>
      <p>
        Open Tally is a demo. If you open an account, we store your name, email,
        a hash of your password, your simulated transactions, and the browser
        and IP address of each sign-in. We set two cookies to keep you signed
        in, and we run no analytics. The company hosting the site may keep
        standard server logs.
      </p>
      <h2>What we&apos;ll collect when accounts open</h2>
      <ul>
        <li>Your name, address, date of birth and Social Security number</li>
        <li>Your account balances and transaction history</li>
        <li>Your income and employment details</li>
        <li>Device and sign-in information</li>
      </ul>
      <h2>How we&apos;ll share it</h2>
      <table>
        <thead>
          <tr>
            <th scope="col">Reason for sharing</th>
            <th scope="col">Do we share?</th>
            <th scope="col">Can you limit it?</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>
              Everyday business, such as processing payments, responding to
              court orders and reporting to regulators
            </td>
            <td>Yes</td>
            <td>No</td>
          </tr>
          <tr>
            <td>Our own marketing</td>
            <td>Yes</td>
            <td>No</td>
          </tr>
          <tr>
            <td>Joint marketing with other financial companies</td>
            <td>No</td>
            <td>We don&apos;t share</td>
          </tr>
          <tr>
            <td>Affiliates&apos; everyday business</td>
            <td>No</td>
            <td>We don&apos;t share</td>
          </tr>
          <tr>
            <td>Marketing by affiliates or non-affiliates</td>
            <td>No</td>
            <td>We don&apos;t share</td>
          </tr>
        </tbody>
      </table>
      <h2>How we&apos;ll protect it</h2>
      <p>
        We&apos;ll use safeguards that comply with federal law, including
        secured systems and limits on which staff can see your information.
      </p>
      <h2>Questions</h2>
      <p>
        Contact us through our <a href={`${REPO_URL}/issues`}>support page</a>.
      </p>
    </>
  );
}
