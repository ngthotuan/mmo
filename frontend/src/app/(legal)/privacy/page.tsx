import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Privacy Policy — AutoContent",
  description: "Privacy Policy for the AutoContent platform.",
};

const LAST_UPDATED = "June 3, 2026";

export default function PrivacyPage() {
  return (
    <article className="space-y-6 text-gray-700 leading-relaxed">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold text-gray-900">Privacy Policy</h1>
        <p className="text-sm text-gray-500">Last updated: {LAST_UPDATED}</p>
      </header>

      <p>
        This Privacy Policy explains how AutoContent (&ldquo;we&rdquo;,
        &ldquo;us&rdquo;, or the &ldquo;Service&rdquo;) collects, uses, and
        protects your information when you use our platform to discover trends,
        generate content, produce videos, and publish them to third-party social
        media platforms such as TikTok, Facebook, and YouTube.
      </p>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          1. Information We Collect
        </h2>
        <ul className="list-disc space-y-1 pl-6">
          <li>
            <strong>Account information</strong> — your name, email address, and
            password (stored in hashed form) when you register.
          </li>
          <li>
            <strong>Connected platform data</strong> — OAuth access tokens and
            basic profile/channel information for the social media accounts you
            choose to connect, used to publish content on your behalf.
          </li>
          <li>
            <strong>Content data</strong> — the scripts, videos, captions, and
            related metadata you create or upload through the Service.
          </li>
          <li>
            <strong>Usage and analytics data</strong> — performance metrics of
            published content and basic logs needed to operate and improve the
            Service.
          </li>
        </ul>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          2. How We Use Your Information
        </h2>
        <ul className="list-disc space-y-1 pl-6">
          <li>to provide, operate, and maintain the Service;</li>
          <li>
            to publish content to the third-party platforms you have connected;
          </li>
          <li>
            to generate scripts and media using AI providers at your direction;
          </li>
          <li>to display analytics about your published content; and</li>
          <li>to secure the Service and comply with legal obligations.</li>
        </ul>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          3. Third-Party Platforms and Services
        </h2>
        <p>
          When you connect a social media account, we interact with that
          platform&rsquo;s API to publish content and retrieve analytics. Your
          use of those platforms is governed by their own privacy policies. We
          also use third-party service providers (for example, AI content
          providers, stock media providers, and cloud storage) to deliver the
          Service; these providers process data only as needed to perform their
          functions.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          4. How We Store and Protect Data
        </h2>
        <p>
          We take reasonable technical and organizational measures to protect
          your information. OAuth tokens for connected accounts are encrypted at
          rest, and passwords are stored using one-way hashing. No method of
          transmission or storage is completely secure, and we cannot guarantee
          absolute security.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          5. Data Retention
        </h2>
        <p>
          We retain your information for as long as your account is active or as
          needed to provide the Service. When you delete your account or
          disconnect a platform, we delete or revoke the associated credentials
          and content within a reasonable period, except where retention is
          required by law.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">6. Your Rights</h2>
        <p>
          Depending on your jurisdiction, you may have the right to access,
          correct, export, or delete your personal information, and to disconnect
          any connected platform at any time from within the Service. To exercise
          these rights, contact us using the details below.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          7. Children&rsquo;s Privacy
        </h2>
        <p>
          The Service is not intended for individuals under the age of 18. We do
          not knowingly collect personal information from children.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          8. Changes to This Policy
        </h2>
        <p>
          We may update this Privacy Policy from time to time. Material changes
          will be reflected by updating the &ldquo;Last updated&rdquo; date
          above. Your continued use of the Service after changes take effect
          constitutes acceptance of the revised policy.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">9. Contact</h2>
        <p>
          If you have questions about this Privacy Policy or your data, contact
          us at{" "}
          <a
            href="mailto:support@autocontent.app"
            className="text-indigo-600 hover:underline"
          >
            support@autocontent.app
          </a>
          .
        </p>
      </section>
    </article>
  );
}
