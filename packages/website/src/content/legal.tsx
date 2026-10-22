import type { LegalDocument } from "@/components/legal-page";

export const privacy: LegalDocument = {
  title: "Privacy policy.",
  summary: "How Burrow handles tunnel traffic and personal information.",
  sections: [
    {
      id: "scope",
      title: "Who we are",
      content: (
        <>
          <p>
            Burrow is operated by Bramble Labs Inc. This policy covers
            useburrow.app, the Burrow tunnel service we operate, and related
            support. Contact{" "}
            <a href="mailto:burrow@mawa.dev">burrow@mawa.dev</a> for privacy
            questions and requests.
          </p>
          <p>
            If you self-host Burrow, you control that installation and its data
            practices. If you visit someone else&apos;s tunnel, the owner of the
            exposed application is responsible for what that application
            collects; their privacy information also applies.
          </p>
        </>
      ),
    },
    {
      id: "traffic",
      title: "Traffic through a tunnel",
      content: (
        <>
          <p>
            Burrow relays HTTP and WebSocket traffic between a public URL and
            the local service its owner exposes. Requests and responses can
            contain URLs, headers, cookies, credentials, form submissions,
            files, and other content chosen by the application and its visitors.
          </p>
          <p>
            The hosted service terminates TLS and processes traffic to relay it.
            It is not end-to-end encryption that hides HTTP content from the
            service operator or its hosting infrastructure. HTTPS protects
            supported network connections; it does not make a public tunnel
            private.
          </p>
          <p>
            Anyone who can reach a tunnel&apos;s public URL can interact with
            the exposed service, subject to that application&apos;s own
            authentication and controls. Burrow owner credentials protect tunnel
            management, not access to your application.
          </p>
        </>
      ),
    },
    {
      id: "records",
      title: "Connection and request information",
      content: (
        <>
          <ul>
            <li>
              <strong>Ownership:</strong> the CLI creates an owner credential
              without requiring an account. It stores credentials locally by
              server origin; the server uses an owner hash to bind and authorize
              tunnels.
            </li>
            <li>
              <strong>Connection records:</strong> tunnel identifier, local
              port, connection/generation state, timestamps, and transfer
              information needed to operate the tunnel.
            </li>
            <li>
              <strong>Request history:</strong> bounded entries containing an
              opaque entry ID, time, method, path, response status, size,
              duration, and network address. The current request-history records
              do not store request/response bodies, authorization headers,
              cookies, or query strings.
            </li>
            <li>
              <strong>Operations and support:</strong> technical logs, security
              information, and messages you send us. Hosting infrastructure may
              process network addresses and request metadata separately from the
              tunnel history.
            </li>
          </ul>
          <p>
            A path can still contain personal or confidential information. Avoid
            embedding secrets in URLs and use test data appropriate for the
            service you expose.
          </p>
        </>
      ),
    },
    {
      id: "purpose",
      title: "Why information is used",
      content: (
        <>
          <p>
            We process this information to connect and route tunnels,
            authenticate management requests, display owner-scoped history and
            status, prevent URL takeover, investigate abuse, troubleshoot
            failures, and provide support. Where a legal basis is required, it
            can include providing the requested service, legitimate
            operational/security interests where permitted, and legal
            obligations.
          </p>
          <p>
            The application does not provide other owners with your management
            history. Server operators and infrastructure providers can process
            information needed to run the service. We do not sell tunnel content
            or use it for advertising.
          </p>
        </>
      ),
    },
    {
      id: "retention",
      title: "Retention and closing a tunnel",
      content: (
        <>
          <p>
            The current server keeps at most 200 request-history entries per
            tunnel. Active connection records have a refreshed 24-hour expiry; a
            disconnected connection&apos;s metadata and history are retained for
            a 20-second reconnection grace period. Closing a tunnel clears its
            history and permanently revokes its identifier.
          </p>
          <p>
            A minimal identifier reservation with its owner hash and closure
            state is retained without automatic expiry to prevent a previously
            shared URL from being reassigned to someone else. This is separate
            from traffic content and request history. Infrastructure logs,
            support records, and any provider backups have their own operational
            retention.
          </p>
          <p>
            You can close a tunnel through the CLI and remove locally stored
            configuration when no longer needed. Contact us about personal
            information, retained ownership records, or provider-held data. Do
            not send the owner credential in an email. A request may require
            proof of control and may be subject to security or legal retention
            exceptions.
          </p>
        </>
      ),
    },
    {
      id: "providers",
      title: "Providers and disclosure",
      content: (
        <>
          <p>
            We use infrastructure, storage, domain and analytics providers to
            operate the hosted service and marketing website. Information is
            processed by these providers to carry out their roles and may be
            processed outside your country.
          </p>
          <p>
            Traffic is also delivered to the tunnel owner&apos;s application and
            any recipients that application selects. We may disclose information
            where necessary for a lawful request, rights or safety, professional
            advice, or a business transfer subject to applicable protections.
          </p>
        </>
      ),
    },
    {
      id: "developer-telemetry",
      title: "CLI and service diagnostics",
      content: (
        <>
          <p>
            Official CLI releases use a separate random installation identifier
            for anonymous command and tunnel outcomes. Analytics receives these
            outcomes; diagnostics receives sanitized error stacks, sampled
            traces and operation metrics and logs. We exclude arguments, ports,
            hostnames, tunnel IDs, credentials, request histories, headers and
            payloads. Browser and CLI identities are not joined.
          </p>
          <p>
            Run <code>burrowctl telemetry disable</code> to opt out and clear
            the local analytics identity, or set <code>DO_NOT_TRACK=1</code> or{" "}
            <code>BURROW_TELEMETRY=0</code>. Running commands honor opt-out
            before uploads. Re-enable with{" "}
            <code>burrowctl telemetry enable</code>; a new identity is created
            on the next invocation. CI and local builds do not initialize
            telemetry. Our hosted server separately reports content-free
            operational diagnostics; self-hosted reporting is off unless the
            operator enables it. Both providers process data in the United
            States. Opt-out does not erase previously received data; contact us
            for information or deletion requests.
          </p>
        </>
      ),
    },
    {
      id: "website",
      title: "Website analytics and demos",
      content: (
        <>
          <p>
            With your permission, advertising partners use cookies and device
            information, including IP addresses and public-page visits, to
            measure advertising and build audiences for ads on other sites.
            Advertising runs only on public website pages after you accept
            advertising cookies. Cookie settings in the footer lets you choose
            analytics and advertising separately, or turn both off. An essential
            first-party cookie stores your choice for 180 days. Optional
            analytics, diagnostics and advertising stay off until you agree.
            Global Privacy Control and Do Not Track prevent them from loading.
            Disabling tracking refreshes the page to unload active tags. Tags
            are excluded from tunnel subdomains; tunnel traffic, credentials and
            form values are not configured as advertising data.
          </p>
          <p>
            With your permission, analytics measures public-page visits,
            install-command copies, product referrals and performance with a
            random browser identifier in local storage. We do not attach names,
            email addresses or tunnel credentials to that identifier. We use
            small website experiments and optional rating surveys. Analytics
            URLs omit queries and fragments; unknown paths are generalized.
          </p>
          <p>
            With your permission, analytics samples masked website recordings at
            10%, retained for 30 days. Text and inputs are masked, media and
            private regions are blocked, and console logs, network headers and
            bodies are excluded. Link attributes are redacted separately.
            Diagnostics receives sanitized website error stacks, sampled
            performance traces and bounded clipboard outcome metrics and logs.
            It does not record website sessions.
          </p>
          <p>
            Disable browser analytics and diagnostics with the website telemetry
            control below. Global Privacy Control and Do Not Track also disable
            browser collection. Opt-out clears the stored analytics identity and
            stops further collection; previously received data is not
            automatically erased. These providers process data in the United
            States. Contact us with information or deletion requests. Server
            diagnostics are separate from this browser preference and exclude
            request content.
          </p>
          <p>
            With your permission, analytics also measures aggregate page visits,
            referrals, and broad device/region information without tracking
            cookies. We do not send tunnel traffic or request-history content as
            website analytics. Performance measurements cover public-page
            loading, responsiveness, and visual stability without tracking
            cookies. Its reported URLs exclude query strings and fragments.
            Hosting can separately process IP addresses, URLs, browser details,
            and request times.
          </p>
          <p>
            The terminal previews contain example data, not live tunnels or
            customer traffic. Their selections stay in the page. Copy controls
            write the displayed text to your clipboard only when selected; they
            do not read your clipboard. Fonts and artwork are served with the
            website.
          </p>
        </>
      ),
    },
    {
      id: "rights",
      title: "Your choices and rights",
      content: (
        <>
          <p>
            You choose which local service to expose and when to close it.
            Protect the exposed application appropriately and tell its visitors
            how their information is used. Self-hosting lets you choose your own
            infrastructure and retention configuration.
          </p>
          <p>
            Depending on your location, you may have rights of access,
            correction, deletion, portability, objection, restriction,
            withdrawal of consent, or complaint to a privacy authority. Contact
            us to exercise these rights; we may verify control and explain
            lawful exceptions. If you believe a child&apos;s personal
            information has been submitted without required authorization, tell
            us.
          </p>
        </>
      ),
    },
    {
      id: "updates",
      title: "Security and updates",
      content: (
        <>
          <p>
            We use authenticated management access, owner-scoped records,
            protected credentials, and encrypted supported connections. No
            service can guarantee complete security. Report suspected exposure
            privately to <a href="mailto:burrow@mawa.dev">burrow@mawa.dev</a>.
          </p>
          <p>
            We update this policy as the service changes, show the effective
            date, and provide notice or obtain consent where required.
          </p>
        </>
      ),
    },
  ],
};

export const terms: LegalDocument = {
  title: "Terms & conditions.",
  summary: "Terms for using the Burrow website and tunnel service.",
  sections: [
    {
      id: "agreement",
      title: "About these terms",
      content: (
        <>
          <p>
            These terms are between you and Bramble Labs Inc. and govern
            Burrow&apos;s website and our hosted tunnel service. By using them,
            you agree to these terms. You must be legally able to accept, obtain
            any required parental or guardian permission, and have authority if
            acting for an organization.
          </p>
          <p>
            The <a href="/privacy">Privacy policy</a> explains information
            handling. Software licences and any separately agreed service terms
            also apply to their respective subject matter.
          </p>
        </>
      ),
    },
    {
      id: "service",
      title: "What a tunnel does",
      content: (
        <>
          <p>
            Burrow gives a local HTTP service a public HTTPS URL and provides
            terminal tools for connection management and request inspection.
            Public URLs are accessible to anyone who can reach them. You are
            responsible for choosing what you expose and for your
            application&apos;s authentication, authorization, content, and
            notices.
          </p>
          <p>
            A tunnel-management credential does not authenticate visitors to the
            exposed application. Burrow is not a private VPN, a backup service,
            or a guarantee that an application is safe to make public. Website
            examples are illustrations and do not create real tunnels.
          </p>
        </>
      ),
    },
    {
      id: "credentials",
      title: "Credentials and your responsibilities",
      content: (
        <>
          <p>
            Keep the CLI&apos;s owner credentials secure, use a trusted server
            origin, and close tunnels you no longer need. You are responsible
            for use you authorize and for having permission to expose a service,
            system, or dataset. Tell us about suspected credential compromise
            without including credentials in public issues or ordinary email.
          </p>
          <p>
            You retain your rights in your application and content. You
            authorize us and our providers to transmit and technically process
            traffic as needed to operate the tunnels you request. This is not a
            transfer of ownership or permission for unrelated advertising use.
          </p>
        </>
      ),
    },
    {
      id: "acceptable-use",
      title: "Acceptable use",
      content: (
        <>
          <p>
            Do not use Burrow to distribute malware, host phishing or fraudulent
            pages, send spam, infringe rights, distribute illegal content,
            expose systems without authorization, bypass access controls, steal
            credentials, or disrupt infrastructure. Do not attempt to claim
            another owner&apos;s tunnel or evade usage and abuse controls.
          </p>
          <p>
            You are responsible for complying with laws and third-party terms
            applicable to your exposed service and visitor data. Report abuse or
            security concerns to{" "}
            <a href="mailto:burrow@mawa.dev">burrow@mawa.dev</a>, including the
            relevant URL and a concise description without unnecessary sensitive
            data.
          </p>
        </>
      ),
    },
    {
      id: "availability",
      title: "Availability and changes",
      content: (
        <>
          <p>
            Burrow is intended for development workflows. Connections can be
            interrupted, expired, revoked, or unavailable; a URL is not a
            promise of permanent hosting or a service-level agreement. Keep your
            local work and important records independently.
          </p>
          <p>
            We may change features, impose reasonable capacity limits, or
            discontinue hosted features with notice where required. Any paid
            offering must disclose its price, renewal, and cancellation terms
            before purchase; using these terms does not itself start a paid
            subscription.
          </p>
        </>
      ),
    },
    {
      id: "open-source",
      title: "Open source and self-hosting",
      content: (
        <>
          <p>
            Burrow&apos;s open-source components are governed by their published
            licences, including the MIT licence where stated. These terms do not
            take away those licence permissions. They govern the website and
            hosted infrastructure we operate.
          </p>
          <p>
            If you self-host, you are responsible for operating, securing, and
            maintaining that deployment, its certificates and storage, and its
            privacy and acceptable-use arrangements. We do not operate
            third-party deployments or the applications tunnel owners expose.
          </p>
        </>
      ),
    },
    {
      id: "ending",
      title: "Suspension and ending use",
      content: (
        <>
          <p>
            You may stop using Burrow and close your tunnels at any time. We may
            restrict or suspend access where reasonably needed to address abuse,
            security, legal requirements, or material violations. We will
            provide notice or a review path where appropriate; urgent risks may
            require immediate action.
          </p>
          <p>
            Closing a tunnel revokes its identifier and clears its request
            history under the Privacy policy. Minimal anti-reassignment records
            and applicable operational records can remain. Contact us if you
            believe a restriction was mistaken.
          </p>
        </>
      ),
    },
    {
      id: "responsibility",
      title: "Responsibility and legal rights",
      content: (
        <>
          <p>
            To the extent permitted by law, Burrow is provided as available
            without guarantees of continuous connectivity, compatibility with
            every local application, or suitability for a particular purpose. We
            remain responsible for obligations and care required by applicable
            law.
          </p>
          <p>
            To the extent permitted by law, we are not liable for indirect or
            consequential loss, lost profits, or independent third-party
            failures. This does not exclude liability that cannot lawfully be
            excluded, including fraud, intentional misconduct, or mandatory
            consumer rights. Nothing requires arbitration or removes a legally
            guaranteed complaint or court remedy.
          </p>
        </>
      ),
    },
    {
      id: "contact",
      title: "Updates and contact",
      content: (
        <>
          <p>
            We may update these terms and will provide notice or obtain renewed
            acceptance where required. If a provision is unenforceable, the rest
            continues to the extent lawful. Ownership, accrued obligations, and
            provisions that reasonably need to survive continue after access
            ends.
          </p>
          <p>
            Contact <a href="mailto:burrow@mawa.dev">burrow@mawa.dev</a> for
            service, legal, privacy, or rights-holder questions.
          </p>
        </>
      ),
    },
  ],
};
