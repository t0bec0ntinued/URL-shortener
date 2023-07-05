CREATE DATABASE test_db
    WITH
    OWNER = test_user
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1
    IS_TEMPLATE = False;
    CREATE TABLE IF NOT EXISTS public.testtb
(
    id integer NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1 ),
    input text COLLATE pg_catalog."default" NOT NULL,
    output text COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT testtb_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.testtb
    OWNER to test_user;
