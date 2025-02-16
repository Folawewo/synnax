from dataclasses import dataclass
from typing import Callable

from .transport import Payload

TYPE_UNKNOWN = "unknown"
TYPE_NONE = "nil"


class ExceptionPayload(Payload):
    """Error payload is a payload that can be sent between a freighter client and server,
    so that it can be decoded into a proper exception by the implementing language.
    """

    type: str | None
    data: str | None


EncoderFunc = Callable[[Exception], str]
DecoderFunc = Callable[[str], Exception]


@dataclass
class _ExceptionProvider:
    encode: EncoderFunc
    decode: DecoderFunc


class _Registry:
    providers: dict[str, _ExceptionProvider]

    def __init__(self):
        self.providers = dict()

    def register(self, _type: str, provider: _ExceptionProvider) -> None:
        if _type in self.providers:
            raise ValueError(f"Error type {_type} is already registered")
        self.providers[_type] = provider

    @staticmethod
    def encode(error: Exception | None) -> ExceptionPayload:
        raise NotImplemented

    def decode(self, encoded: ExceptionPayload) -> Exception | None:
        assert isinstance(encoded, ExceptionPayload)
        if encoded.type == TYPE_NONE:
            return None
        if encoded.type in self.providers:
            if encoded.data is None:
                raise Exception(f"Error data is missing for {encoded.type}")
            return self.providers[encoded.type].decode(encoded.data)
        return Exception(encoded.data)


REGISTRY = _Registry()


def register_exception(_type: str, _encode: EncoderFunc, _decode: DecoderFunc) -> None:
    """Registers a custom error type with the freighter error registry, which allows
    it to be sent over the network.

    :param _type: The type of the error, which must be unique.
    :param _encode: A function that takes an exception and returns a string.
    :param _decode: A function that takes a string and returns an exception.
    :return: None
    """
    REGISTRY.register(_type, _ExceptionProvider(_encode, _decode))


def encode_expection(exc: Exception) -> ExceptionPayload:
    """Encodes an exception into a payload that can be sent between a freighter server
    and client.

    :param exc: The exception to encode.
    :return: The encoded error payload.
    """
    return REGISTRY.encode(exc)


def decode_exception(encoded: ExceptionPayload) -> Exception | None:
    """Decode decodes an error payload into an exception. If a custom decoder can be found
    for the error type, it will be used. Otherwise, a generic Exception containing the
    error data is returned.

    :param encoded: The encoded error payload.
    :return: The decoded exception.
    """
    return REGISTRY.decode(encoded)


class Unreachable(Exception):
    """
    Raise when a target is unreachable.
    """

    target: str
    message: str

    def __init(self, target: str, message="freighter.errors.Unreachable"):
        self.target = target
        self.message = message
        super().__init__(message)

    def __str__(self):
        return self.message


class StreamClosed(Exception):
    """
    Raised when a stream is closed.
    """

    def __str__(self):
        return "freighter.errors.StreamClosed"


class EOF(Exception):
    """
    Raised when a stream is closed.
    """

    def __str__(self):
        return "freighter.errors.EOF"


_EXCEPTIONS = [
    Unreachable,
    StreamClosed,
    EOF,
]


def _freighter_encode(exc: Exception) -> str:
    if isinstance(exc, Unreachable):
        return "Unreachable"
    if isinstance(exc, StreamClosed):
        return "StreamClosed"
    if isinstance(exc, EOF):
        return "EOF"

    raise ValueError(f"Unknown freighter exception: {exc}")


def _freighter_decode(exc: str) -> Exception:
    if exc == "Unreachable":
        return Unreachable()
    if exc == "StreamClosed":
        return StreamClosed()
    if exc == "EOF":
        return EOF()
    raise ValueError(f"Unknown freighter exception: {exc}")


_FREIGHTER_TYPE = "freighter"
register_exception(_FREIGHTER_TYPE, _freighter_encode, _freighter_decode)
