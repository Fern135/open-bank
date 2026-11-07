import type { Metadata } from "next";

export const metadata: Metadata = { title: "Fee schedule" };

export default function Fees() {
  return (
    <>
      <h1>Fee schedule</h1>
      <p>Last updated November 7, 2026</p>
      <p>
        We haven&apos;t set our fees yet. We&apos;ll publish them here before we
        open any account.
      </p>
      <table>
        <thead>
          <tr>
            <th scope="col">Service</th>
            <th scope="col">Fee</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Monthly maintenance</td>
            <td>Not set</td>
          </tr>
          <tr>
            <td>Outgoing domestic wire</td>
            <td>Not set</td>
          </tr>
          <tr>
            <td>Outgoing international wire</td>
            <td>Not set</td>
          </tr>
          <tr>
            <td>Out-of-network ATM</td>
            <td>Not set</td>
          </tr>
          <tr>
            <td>Paper statement</td>
            <td>Not set</td>
          </tr>
          <tr>
            <td>Stop payment</td>
            <td>Not set</td>
          </tr>
        </tbody>
      </table>
      <p>
        We don&apos;t plan to charge overdraft fees, because we decline a
        payment your available balance can&apos;t cover.
      </p>
    </>
  );
}
