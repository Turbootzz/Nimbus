FROM postgres:18-alpine

COPY entrypoint.sh /usr/local/bin/nimbus-entrypoint.sh
RUN chmod +x /usr/local/bin/nimbus-entrypoint.sh

ENTRYPOINT ["nimbus-entrypoint.sh"]
CMD ["postgres"]
