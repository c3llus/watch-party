package movie

// movieService provides movie-related services.
type movieService struct {
}

// NewMovieService creates a new user service instance.
func NewMovieService() *movieService {
	return &movieService{}
}
