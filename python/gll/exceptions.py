"""Exception classes for the gll package."""


class GllError(Exception):
    """Base exception for GLL parsing errors."""

    pass


class ParseError(GllError):
    """Error during GLL file parsing."""

    pass


class ResourceError(GllError):
    """Error extracting a resource from a GLL file."""

    pass


class ConfigurationError(GllError):
    """Error in array configuration."""

    pass
