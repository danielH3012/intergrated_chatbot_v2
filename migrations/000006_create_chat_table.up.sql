-- 000006_create_chat_table.up.sql
-- Table: public.chat (Conversation logs and audit trails)

CREATE TABLE IF NOT EXISTS public.chat (
    id VARCHAR(125) CONSTRAINT chat_chat_id_not_null NOT NULL,
    "user" BOOLEAN,
    chat TEXT,
    created_at VARCHAR(125),
    user_id BIGINT,
    username VARCHAR(125),
    role VARCHAR(125),
    models VARCHAR(125),
    attachment_name VARCHAR(255),
    attachment_url TEXT,
    attachment_type VARCHAR(100),
    attachment_size BIGINT,
    attachment_text TEXT,
    CONSTRAINT chat_pkey PRIMARY KEY (id),
    CONSTRAINT fk_chat_user_id FOREIGN KEY (user_id) REFERENCES public."user"(user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_username ON public.chat (username);
CREATE INDEX IF NOT EXISTS idx_chat_created_at ON public.chat (created_at);
