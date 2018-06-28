var gulp = require('gulp');

var coffee = require('gulp-coffee');
var pug = require('gulp-pug');
var gzip = require('gulp-zopfli-green');
var cleanCSS = require('gulp-clean-css');

var paths = {
    coffee: 'client/**/*.coffee',
    pages:  'templates/**/*.pug'
};

gulp.task('coffee', function() {
    return gulp.src(paths.coffee)
        .pipe(coffee())
        .pipe(gulp.dest('client'));
});

gulp.task('pages', function() {
    return gulp.src(paths.pages)
        .pipe(pug())
        .pipe(gulp.dest('templates'));
});

gulp.task('css', function() {
    return gulp.src('res/css/main.css')
        .pipe(cleanCSS())
        .pipe(gulp.dest('build/css'))
});

gulp.task('copy_fonts', function() {
    return gulp.src('res/fonts/glyphicons-halflings-regular.*')
        .pipe(gulp.dest('serve/fonts'));
});

gulp.task('compress', gulp.series(gulp.parallel(
    'coffee',
    'css',
), function() {
    return gulp.src([
            'build/**/*',
            'client/*.js',
            'res/**/*.min.js',
            'res/**/ng-google-chart.js',
            'res/**/ngClip.js',
            'res/**/bootstrap.min.css',
            'res/favicon.ico'
        ])
        .pipe(gzip({ append: false }))
        .pipe(gulp.dest('serve'));
}));

gulp.task('build', gulp.parallel(
    'pages', 
    'compress',
    'copy_fonts'
));

gulp.task('watch', gulp.series('build', function() {
    gulp.watch(paths.coffee).on('change', gulp.series('coffee'));
    gulp.watch(paths.pages).on('change', gulp.series('pages'));
}));

gulp.task('default', gulp.series('build'));
