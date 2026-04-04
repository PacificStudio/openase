env "ent" {
  src = "ent://ent/schema"
  dev = "docker://postgres/16/dev?search_path=public"
  migration {
    dir = "file://ent/migrate/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
