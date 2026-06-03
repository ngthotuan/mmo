import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Terms of Service — AutoContent",
  description: "Terms of Service for the AutoContent platform.",
};

const LAST_UPDATED = "June 3, 2026";

export default function TermsPage() {
  return (
    <article className="space-y-6 text-gray-700 leading-relaxed">
      <header className="space-y-2">
        <h1 className="text-3xl font-bold text-gray-900">Terms of Service</h1>
        <p className="text-sm text-gray-500">Last updated: {LAST_UPDATED}</p>
      </header>

      <p>
        These Terms of Service (&ldquo;Terms&rdquo;) govern your access to and
        use of AutoContent (the &ldquo;Service&rdquo;), a platform for
        discovering trends, generating content, producing videos, and publishing
        them to third-party social media platforms such as TikTok, Facebook, and
        YouTube. By accessing or using the Service, you agree to be bound by
        these Terms. If you do not agree, do not use the Service.
      </p>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          1. Eligibility and Accounts
        </h2>
        <p>
          You must be at least 18 years old and capable of forming a binding
          contract to use the Service. You are responsible for maintaining the
          confidentiality of your account credentials and for all activity that
          occurs under your account. Notify us immediately of any unauthorized
          use.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          2. Connected Third-Party Accounts
        </h2>
        <p>
          The Service lets you connect third-party social media accounts to
          publish content on your behalf. You authorize us to access and use
          those accounts solely to provide the Service. Your use of each
          third-party platform remains subject to that platform&rsquo;s own terms
          and policies, and you are responsible for complying with them.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          3. Your Content
        </h2>
        <p>
          You retain ownership of the content you create, upload, or publish
          through the Service. You grant us a limited license to store, process,
          and transmit your content as necessary to operate the Service. You are
          solely responsible for the content you publish and for ensuring you
          hold all rights necessary to do so.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          4. Acceptable Use
        </h2>
        <p>You agree not to use the Service to:</p>
        <ul className="list-disc space-y-1 pl-6">
          <li>
            publish content that is unlawful, infringing, deceptive, or harmful;
          </li>
          <li>
            violate the terms, rate limits, or automation policies of any
            connected platform;
          </li>
          <li>
            spam, mislead audiences, or artificially manipulate engagement; or
          </li>
          <li>
            attempt to disrupt, reverse engineer, or gain unauthorized access to
            the Service.
          </li>
        </ul>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          5. AI-Generated Content
        </h2>
        <p>
          The Service uses artificial intelligence to assist in generating
          scripts and media. AI output may be inaccurate or unsuitable, and you
          are responsible for reviewing and approving content before it is
          published.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          6. Disclaimers
        </h2>
        <p>
          The Service is provided &ldquo;as is&rdquo; and &ldquo;as
          available&rdquo; without warranties of any kind, whether express or
          implied. We do not guarantee that the Service will be uninterrupted,
          error-free, or that published content will achieve any particular
          result.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          7. Limitation of Liability
        </h2>
        <p>
          To the maximum extent permitted by law, we will not be liable for any
          indirect, incidental, special, consequential, or punitive damages, or
          for any loss of profits, data, or goodwill arising from your use of the
          Service.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          8. Termination
        </h2>
        <p>
          We may suspend or terminate your access to the Service at any time if
          you violate these Terms or if we discontinue the Service. You may stop
          using the Service at any time.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">
          9. Changes to These Terms
        </h2>
        <p>
          We may update these Terms from time to time. Material changes will be
          reflected by updating the &ldquo;Last updated&rdquo; date above. Your
          continued use of the Service after changes take effect constitutes
          acceptance of the revised Terms.
        </p>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold text-gray-900">10. Contact</h2>
        <p>
          If you have questions about these Terms, contact us at{" "}
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
