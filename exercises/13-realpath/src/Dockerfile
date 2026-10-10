FROM filesystems-101-exercises:0

RUN mkdir /dir
RUN touch /file
RUN touch /.dot
RUN ln -s "file" /symlink0
RUN ln -s "./dir" /symlink1
RUN ln -s "/file" /symlink2
RUN ln -s "dir/../symlink2" /symlink3

COPY ./a.out /
WORKDIR /
