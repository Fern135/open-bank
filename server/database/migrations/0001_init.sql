CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name              text NOT NULL,
    email             text NOT NULL UNIQUE,
    password_hash     text NOT NULL,
    totp_secret       bytea,
    totp_enabled      boolean NOT NULL DEFAULT false,
    totp_last_counter bigint NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               uuid NOT NULL REFERENCES users,
    refresh_hash          bytea NOT NULL,
    previous_refresh_hash bytea,
    rotated_at            timestamptz,
    user_agent            text NOT NULL DEFAULT '',
    ip                    text NOT NULL DEFAULT '',
    created_at            timestamptz NOT NULL DEFAULT now(),
    last_used_at          timestamptz NOT NULL DEFAULT now(),
    expires_at            timestamptz NOT NULL,
    revoked_at            timestamptz
);
CREATE INDEX sessions_user_id ON sessions (user_id);

CREATE TABLE accounts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid REFERENCES users,
    number     text NOT NULL UNIQUE,
    type       text NOT NULL CHECK (type IN ('checking', 'savings', 'funding')),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- only the bank's own funding account has no customer
    CHECK ((type = 'funding') = (user_id IS NULL))
);
CREATE INDEX accounts_user_id ON accounts (user_id);

-- simulated deposits are posted against this account, so the books still balance
INSERT INTO accounts (id, number, type, name)
VALUES ('00000000-0000-0000-0000-000000000001', '0000000000', 'funding', 'Open Tally funding');

CREATE TABLE transactions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind            text NOT NULL CHECK (kind IN ('deposit', 'internal_transfer', 'external_transfer')),
    memo            text NOT NULL DEFAULT '',
    initiated_by    uuid NOT NULL REFERENCES users,
    idempotency_key text NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (initiated_by, idempotency_key)
);

-- amount_cents is signed: positive credits an account, negative debits it.
-- A balance is always the sum of its entries, it is never stored.
CREATE TABLE entries (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transaction_id uuid NOT NULL REFERENCES transactions,
    account_id     uuid NOT NULL REFERENCES accounts,
    amount_cents   bigint NOT NULL CHECK (amount_cents <> 0),
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX entries_account_id ON entries (account_id, id);
CREATE INDEX entries_transaction_id ON entries (transaction_id);

CREATE TABLE audit_log (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users,
    action     text NOT NULL,
    ip         text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_user_id ON audit_log (user_id, id);

-- The ledger and audit log are append-only: a mistake gets a new entry that reverses it.
CREATE FUNCTION forbid_change() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION '% is append-only', TG_TABLE_NAME;
END $$;

CREATE TRIGGER entries_append_only BEFORE UPDATE OR DELETE ON entries
    FOR EACH ROW EXECUTE FUNCTION forbid_change();
CREATE TRIGGER entries_no_truncate BEFORE TRUNCATE ON entries
    FOR EACH STATEMENT EXECUTE FUNCTION forbid_change();
CREATE TRIGGER transactions_append_only BEFORE UPDATE OR DELETE ON transactions
    FOR EACH ROW EXECUTE FUNCTION forbid_change();
CREATE TRIGGER transactions_no_truncate BEFORE TRUNCATE ON transactions
    FOR EACH STATEMENT EXECUTE FUNCTION forbid_change();
CREATE TRIGGER audit_log_append_only BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION forbid_change();
CREATE TRIGGER audit_log_no_truncate BEFORE TRUNCATE ON audit_log
    FOR EACH STATEMENT EXECUTE FUNCTION forbid_change();

-- Runs at commit, so a transaction that doesn't sum to zero can never be saved.
CREATE FUNCTION check_balanced() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF (SELECT sum(amount_cents) FROM entries WHERE transaction_id = NEW.transaction_id) <> 0 THEN
        RAISE EXCEPTION 'transaction % does not balance', NEW.transaction_id;
    END IF;
    RETURN NULL;
END $$;

CREATE CONSTRAINT TRIGGER entries_balanced AFTER INSERT ON entries
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_balanced();
