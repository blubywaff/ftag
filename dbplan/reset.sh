rm fe/*
( . ../pytools/venv/bin/activate && python fullexport.py -c ../ftag.config.json)
python3 jsontosql.py
psql -h localhost -U postgres -f table.sql; for i in $(ls fe); do psql -h localhost -U postgres -f fe/$i; done
